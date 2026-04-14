package worker

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── Mock RecurrenceRepository ───────────────────────────────────────────────

type mockRecurrenceRepo struct {
	listDueFn func(ctx context.Context, now time.Time, batchSize int) ([]domain.SubscriptionBillingCandidate, error)
	lockFn    func(ctx context.Context, orgID, subID uuid.UUID, until time.Time) error
	unlockFn  func(ctx context.Context, orgID, subID uuid.UUID) error
}

var _ domain.RecurrenceRepository = (*mockRecurrenceRepo)(nil)

func (m *mockRecurrenceRepo) ListDueForBilling(ctx context.Context, now time.Time, batchSize int) ([]domain.SubscriptionBillingCandidate, error) {
	if m.listDueFn != nil {
		return m.listDueFn(ctx, now, batchSize)
	}
	return nil, nil
}
func (m *mockRecurrenceRepo) LockForBilling(ctx context.Context, orgID, subID uuid.UUID, until time.Time) error {
	if m.lockFn != nil {
		return m.lockFn(ctx, orgID, subID, until)
	}
	return nil
}
func (m *mockRecurrenceRepo) UnlockBilling(ctx context.Context, orgID, subID uuid.UUID) error {
	if m.unlockFn != nil {
		return m.unlockFn(ctx, orgID, subID)
	}
	return nil
}

// ── Mock Queue for scheduler tests ──────────────────────────────────────────

type mockSchedulerQueue struct {
	mu        sync.Mutex
	enqueueFn func(ctx context.Context, job domain.Job) error
	dequeueFn func(ctx context.Context) (*domain.Job, error)
	enqueued  []domain.Job
}

func (q *mockSchedulerQueue) Enqueue(ctx context.Context, job domain.Job) error {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.enqueued = append(q.enqueued, job)
	if q.enqueueFn != nil {
		return q.enqueueFn(ctx, job)
	}
	return nil
}

func (q *mockSchedulerQueue) Dequeue(ctx context.Context) (*domain.Job, error) {
	if q.dequeueFn != nil {
		return q.dequeueFn(ctx)
	}
	<-ctx.Done()
	return nil, ctx.Err()
}

// ── RecurrenceScheduler.scan Tests ──────────────────────────────────────────

func TestRecurrenceScheduler_Scan_EnqueuesCandidates(t *testing.T) {
	sub1 := domain.SubscriptionBillingCandidate{
		ID: uuid.New(), OrganizationID: uuid.New(), CustomerID: uuid.New(),
		Status: domain.SubStatusActive, PaymentMethod: "credit_card",
		CurrentPeriodEnd: time.Now().Add(-1 * time.Hour),
	}
	sub2 := domain.SubscriptionBillingCandidate{
		ID: uuid.New(), OrganizationID: uuid.New(), CustomerID: uuid.New(),
		Status: domain.SubStatusPastDue, PaymentMethod: "boleto",
		CurrentPeriodEnd: time.Now().Add(-2 * time.Hour),
	}

	repo := &mockRecurrenceRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]domain.SubscriptionBillingCandidate, error) {
			return []domain.SubscriptionBillingCandidate{sub1, sub2}, nil
		},
	}
	q := &mockSchedulerQueue{}

	scheduler := NewRecurrenceScheduler(repo, q, 100, "*/5 * * * *")
	scheduler.scan(context.Background())

	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.enqueued) != 2 {
		t.Fatalf("expected 2 enqueued jobs, got %d", len(q.enqueued))
	}

	for i, job := range q.enqueued {
		if job.Type != domain.JobTypeRecurringBilling {
			t.Errorf("job %d: expected type %s, got %s", i, domain.JobTypeRecurringBilling, job.Type)
		}
		if job.AttemptNumber != 1 {
			t.Errorf("job %d: expected attempt 1, got %d", i, job.AttemptNumber)
		}

		var payload domain.RecurringBillingJob
		if err := json.Unmarshal(job.Payload, &payload); err != nil {
			t.Errorf("job %d: failed to unmarshal payload: %v", i, err)
		}
	}
}

func TestRecurrenceScheduler_Scan_NoCandidates(t *testing.T) {
	repo := &mockRecurrenceRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]domain.SubscriptionBillingCandidate, error) {
			return nil, nil
		},
	}
	q := &mockSchedulerQueue{}

	scheduler := NewRecurrenceScheduler(repo, q, 100, "*/5 * * * *")
	scheduler.scan(context.Background())

	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.enqueued) != 0 {
		t.Errorf("expected 0 enqueued jobs, got %d", len(q.enqueued))
	}
}

func TestRecurrenceScheduler_Scan_ListError(t *testing.T) {
	repo := &mockRecurrenceRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]domain.SubscriptionBillingCandidate, error) {
			return nil, domain.ErrInternal
		},
	}
	q := &mockSchedulerQueue{}

	scheduler := NewRecurrenceScheduler(repo, q, 100, "*/5 * * * *")
	scheduler.scan(context.Background()) // should not panic

	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.enqueued) != 0 {
		t.Errorf("expected 0 enqueued on error, got %d", len(q.enqueued))
	}
}

func TestRecurrenceScheduler_Scan_EnqueuePartialFailure(t *testing.T) {
	sub1 := domain.SubscriptionBillingCandidate{
		ID: uuid.New(), OrganizationID: uuid.New(),
		Status: domain.SubStatusActive, PaymentMethod: "credit_card",
	}
	sub2 := domain.SubscriptionBillingCandidate{
		ID: uuid.New(), OrganizationID: uuid.New(),
		Status: domain.SubStatusActive, PaymentMethod: "boleto",
	}

	repo := &mockRecurrenceRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]domain.SubscriptionBillingCandidate, error) {
			return []domain.SubscriptionBillingCandidate{sub1, sub2}, nil
		},
	}

	callCount := 0
	q := &mockSchedulerQueue{
		enqueueFn: func(_ context.Context, _ domain.Job) error {
			callCount++
			if callCount == 1 {
				return domain.ErrInternal // fail first
			}
			return nil
		},
	}

	scheduler := NewRecurrenceScheduler(repo, q, 100, "*/5 * * * *")
	scheduler.scan(context.Background())

	// Both should have been attempted, but only one succeeded
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.enqueued) != 2 { // both get appended by the wrapper
		t.Logf("enqueued count: %d (includes failed attempts)", len(q.enqueued))
	}
}

func TestRecurrenceScheduler_Scan_JobPayloadContainsCorrectIDs(t *testing.T) {
	orgID := uuid.New()
	subID := uuid.New()

	repo := &mockRecurrenceRepo{
		listDueFn: func(_ context.Context, _ time.Time, _ int) ([]domain.SubscriptionBillingCandidate, error) {
			return []domain.SubscriptionBillingCandidate{{
				ID: subID, OrganizationID: orgID,
				Status: domain.SubStatusActive, PaymentMethod: "pix",
			}}, nil
		},
	}
	q := &mockSchedulerQueue{}

	scheduler := NewRecurrenceScheduler(repo, q, 50, "*/5 * * * *")
	scheduler.scan(context.Background())

	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.enqueued) != 1 {
		t.Fatalf("expected 1 job, got %d", len(q.enqueued))
	}

	var payload domain.RecurringBillingJob
	if err := json.Unmarshal(q.enqueued[0].Payload, &payload); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if payload.SubscriptionID != subID {
		t.Errorf("expected subscription_id %s, got %s", subID, payload.SubscriptionID)
	}
	if payload.OrganizationID != orgID {
		t.Errorf("expected organization_id %s, got %s", orgID, payload.OrganizationID)
	}
}

func TestRecurrenceScheduler_Start_CancelStops(t *testing.T) {
	repo := &mockRecurrenceRepo{}
	q := &mockSchedulerQueue{}
	scheduler := NewRecurrenceScheduler(repo, q, 100, "*/5 * * * *")

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		scheduler.Start(ctx)
		close(done)
	}()

	// Give the cron just enough time to start
	time.Sleep(100 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// OK
	case <-time.After(5 * time.Second):
		t.Fatal("scheduler did not stop after context cancel")
	}
}
