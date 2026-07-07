package worker

import (
	"context"
	"testing"
	"time"
)

// --- Mock types ---

type mockCleanupRepo struct {
	deleteCredentialLogsFn  func(ctx context.Context, cutoff time.Time) (int64, error)
	deleteWebhookLogsFn     func(ctx context.Context, cutoff time.Time) (int64, error)
	deleteIdempotencyKeysFn func(ctx context.Context) (int64, error)
}

func (m *mockCleanupRepo) DeleteCredentialLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return m.deleteCredentialLogsFn(ctx, cutoff)
}

func (m *mockCleanupRepo) DeleteWebhookLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	return m.deleteWebhookLogsFn(ctx, cutoff)
}

func (m *mockCleanupRepo) DeleteExpiredIdempotencyKeys(ctx context.Context) (int64, error) {
	return m.deleteIdempotencyKeysFn(ctx)
}

// --- Tests ---

func TestCleanLogs_DeletesBothTablesWithCorrectCutoff(t *testing.T) {
	fixedNow := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	expectedCutoff := fixedNow.Add(-90 * 24 * time.Hour)

	var capturedCredCutoff, capturedWebhookCutoff time.Time

	repo := &mockCleanupRepo{
		deleteCredentialLogsFn: func(_ context.Context, cutoff time.Time) (int64, error) {
			capturedCredCutoff = cutoff
			return 42, nil
		},
		deleteWebhookLogsFn: func(_ context.Context, cutoff time.Time) (int64, error) {
			capturedWebhookCutoff = cutoff
			return 10, nil
		},
	}

	w := NewCleanupWorker(repo)
	w.now = func() time.Time { return fixedNow }

	w.cleanLogs(context.Background())

	if !capturedCredCutoff.Equal(expectedCutoff) {
		t.Errorf("credential logs cutoff = %v, want %v", capturedCredCutoff, expectedCutoff)
	}
	if !capturedWebhookCutoff.Equal(expectedCutoff) {
		t.Errorf("webhook logs cutoff = %v, want %v", capturedWebhookCutoff, expectedCutoff)
	}
}

func TestCleanLogs_ContinuesOnCredentialError(t *testing.T) {
	var webhookCalled bool

	repo := &mockCleanupRepo{
		deleteCredentialLogsFn: func(_ context.Context, _ time.Time) (int64, error) {
			return 0, context.DeadlineExceeded
		},
		deleteWebhookLogsFn: func(_ context.Context, _ time.Time) (int64, error) {
			webhookCalled = true
			return 5, nil
		},
	}

	w := NewCleanupWorker(repo)
	w.now = time.Now

	w.cleanLogs(context.Background())

	if !webhookCalled {
		t.Error("expected webhook logs cleanup to be called even when credential logs cleanup fails")
	}
}

func TestCleanIdempotencyKeys_CallsRepo(t *testing.T) {
	var called bool

	repo := &mockCleanupRepo{
		deleteIdempotencyKeysFn: func(_ context.Context) (int64, error) {
			called = true
			return 100, nil
		},
	}

	w := NewCleanupWorker(repo)
	w.cleanIdempotencyKeys(context.Background())

	if !called {
		t.Error("expected DeleteExpiredIdempotencyKeys to be called")
	}
}

func TestStart_RespectsContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	repo := &mockCleanupRepo{
		deleteCredentialLogsFn:  func(_ context.Context, _ time.Time) (int64, error) { return 0, nil },
		deleteWebhookLogsFn:     func(_ context.Context, _ time.Time) (int64, error) { return 0, nil },
		deleteIdempotencyKeysFn: func(_ context.Context) (int64, error) { return 0, nil },
	}

	w := NewCleanupWorker(repo)

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	select {
	case <-done:
		// success: Start exited after context cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not exit after context cancellation")
	}
}
