package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"i9pay/internal/domain"
	"i9pay/internal/queue"

	"github.com/google/uuid"
)

// ── mock repos for dispatcher tests ───────────────────────────────────────────

// Task 11.1: Mocks updated for the new WebhookDispatcher signature
// (no logRepo, no httpClient; uses queue.Queue).

type mockEndpointRepo struct {
	listActiveByEventFn func(ctx context.Context, orgID uuid.UUID, eventType string, env string) ([]domain.WebhookEndpoint, error)
}

var _ domain.WebhookEndpointRepository = (*mockEndpointRepo)(nil)

func (m *mockEndpointRepo) Create(_ context.Context, _ uuid.UUID, _ domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockEndpointRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockEndpointRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockEndpointRepo) Update(_ context.Context, _, _ uuid.UUID, _ domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockEndpointRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockEndpointRepo) ListActiveByEvent(ctx context.Context, orgID uuid.UUID, eventType, env string) ([]domain.WebhookEndpoint, error) {
	if m.listActiveByEventFn != nil {
		return m.listActiveByEventFn(ctx, orgID, eventType, env)
	}
	return nil, nil
}

type mockDispatcherSecretReader struct {
	getWebhookSecretFn func(ctx context.Context, orgID uuid.UUID) (string, error)
}

var _ domain.OrganizationSecretReader = (*mockDispatcherSecretReader)(nil)

func (m *mockDispatcherSecretReader) GetWebhookSecret(ctx context.Context, orgID uuid.UUID) (string, error) {
	if m.getWebhookSecretFn != nil {
		return m.getWebhookSecretFn(ctx, orgID)
	}
	return "test-secret", nil
}

type mockQueue struct {
	mu        sync.Mutex
	enqueueFn func(ctx context.Context, job queue.WebhookJob) error
	dequeueFn func(ctx context.Context) (*queue.WebhookJob, error)
	jobs      []queue.WebhookJob
	calls     int
}

var _ queue.Queue = (*mockQueue)(nil)

func (m *mockQueue) Enqueue(ctx context.Context, job queue.WebhookJob) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls++
	if m.enqueueFn != nil {
		err := m.enqueueFn(ctx, job)
		if err == nil {
			m.jobs = append(m.jobs, job)
		}
		return err
	}
	m.jobs = append(m.jobs, job)
	return nil
}

func (m *mockQueue) Dequeue(ctx context.Context) (*queue.WebhookJob, error) {
	if m.dequeueFn != nil {
		return m.dequeueFn(ctx)
	}
	return nil, errors.New("not implemented")
}

// ── Task 11.1: NewWebhookDispatcher construction ─────────────────────────────

func TestNewWebhookDispatcher_Construction(t *testing.T) {
	repo := &mockEndpointRepo{}
	secret := &mockDispatcherSecretReader{}
	q := &mockQueue{}

	d := NewWebhookDispatcher(repo, secret, q)

	if d == nil {
		t.Fatal("expected non-nil dispatcher")
		return
	}
	if d.endpointRepo == nil {
		t.Fatal("expected endpointRepo to be set")
	}
	if d.secretReader == nil {
		t.Fatal("expected secretReader to be set")
	}
	if d.queue == nil {
		t.Fatal("expected queue to be set")
	}
}

// ── Task 11.2: Dispatch with zero active endpoints enqueues no jobs ──────────

func TestDispatch_ZeroEndpoints_NoJobsEnqueued(t *testing.T) {
	orgID := uuid.New()
	q := &mockQueue{}

	d := NewWebhookDispatcher(
		&mockEndpointRepo{
			listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
				return []domain.WebhookEndpoint{}, nil
			},
		},
		&mockDispatcherSecretReader{
			getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "valid-secret", nil
			},
		},
		q,
	)

	err := d.Dispatch(context.Background(), orgID, "customer.created", "production", []byte(`{"id":"1"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q.mu.Lock()
	defer q.mu.Unlock()
	if q.calls != 0 {
		t.Fatalf("expected 0 enqueue calls, got %d", q.calls)
	}
}

// ── Task 11.3: Dispatch with N endpoints enqueues exactly N jobs ─────────────

func TestDispatch_NEndpoints_EnqueuesNJobs(t *testing.T) {
	orgID := uuid.New()
	endpoints := []domain.WebhookEndpoint{
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook1"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook2"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook3"},
	}

	q := &mockQueue{}

	d := NewWebhookDispatcher(
		&mockEndpointRepo{
			listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
				return endpoints, nil
			},
		},
		&mockDispatcherSecretReader{
			getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "test-secret", nil
			},
		},
		q,
	)

	err := d.Dispatch(context.Background(), orgID, "customer.created", "production", []byte(`{"id":"1"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) != 3 {
		t.Fatalf("expected 3 enqueued jobs, got %d", len(q.jobs))
	}

	for i, job := range q.jobs {
		if job.Secret != "test-secret" {
			t.Errorf("job[%d]: expected Secret='test-secret', got %q", i, job.Secret)
		}
		if job.EndpointURL != endpoints[i].URL {
			t.Errorf("job[%d]: expected EndpointURL=%q, got %q", i, endpoints[i].URL, job.EndpointURL)
		}
		if job.EndpointID != endpoints[i].ID {
			t.Errorf("job[%d]: expected EndpointID=%s, got %s", i, endpoints[i].ID, job.EndpointID)
		}
	}
}

// ── Task 11.4: Dispatch continues enqueuing when Enqueue fails for a subset ──

func TestDispatch_EnqueuePartialFailure_ContinuesAll(t *testing.T) {
	orgID := uuid.New()
	endpoints := []domain.WebhookEndpoint{
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook1"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook2"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook3"},
	}

	callCount := 0
	q := &mockQueue{
		enqueueFn: func(_ context.Context, _ queue.WebhookJob) error {
			callCount++
			if callCount == 2 {
				return errors.New("queue temporarily unavailable")
			}
			return nil
		},
	}

	d := NewWebhookDispatcher(
		&mockEndpointRepo{
			listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
				return endpoints, nil
			},
		},
		&mockDispatcherSecretReader{
			getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "test-secret", nil
			},
		},
		q,
	)

	err := d.Dispatch(context.Background(), orgID, "customer.created", "production", []byte(`{"id":"1"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 enqueue attempts must have been made, even though the 2nd failed
	q.mu.Lock()
	defer q.mu.Unlock()
	if q.calls != 3 {
		t.Fatalf("expected 3 enqueue attempts, got %d", q.calls)
	}
}

// ── Task 11.5: No-op — old deliver/deliverWithRetry/sign tests removed ───────
// The methods deliver, deliverWithRetry, and sign have been moved to WebhookWorker.
// There were no pre-existing tests for these methods in the new dispatcher;
// the old tests that referenced them have been removed as part of this refactor.
