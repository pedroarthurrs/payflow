package worker

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── mock repos for scheduler tests ────────────────────────────────────────────

type mockSchedulerEndpointRepo struct {
	listActiveByEventFn func(ctx context.Context, orgID uuid.UUID, eventType string, env string) ([]domain.WebhookEndpoint, error)
}

var _ domain.WebhookEndpointRepository = (*mockSchedulerEndpointRepo)(nil)

func (m *mockSchedulerEndpointRepo) Create(_ context.Context, _ uuid.UUID, _ domain.CreateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockSchedulerEndpointRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockSchedulerEndpointRepo) List(_ context.Context, _ uuid.UUID, _ domain.ListParams) ([]domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockSchedulerEndpointRepo) Update(_ context.Context, _, _ uuid.UUID, _ domain.UpdateWebhookEndpointInput) (*domain.WebhookEndpoint, error) {
	return nil, nil
}
func (m *mockSchedulerEndpointRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockSchedulerEndpointRepo) ListActiveByEvent(ctx context.Context, orgID uuid.UUID, eventType, env string) ([]domain.WebhookEndpoint, error) {
	if m.listActiveByEventFn != nil {
		return m.listActiveByEventFn(ctx, orgID, eventType, env)
	}
	return nil, nil
}

type mockSchedulerSecretReader struct {
	getWebhookSecretFn func(ctx context.Context, orgID uuid.UUID) (string, error)
}

var _ domain.OrganizationSecretReader = (*mockSchedulerSecretReader)(nil)

func (m *mockSchedulerSecretReader) GetWebhookSecret(ctx context.Context, orgID uuid.UUID) (string, error) {
	if m.getWebhookSecretFn != nil {
		return m.getWebhookSecretFn(ctx, orgID)
	}
	return "test-secret", nil
}

type mockWebhookSchedulerQueue struct {
	mu        sync.Mutex
	enqueueFn func(ctx context.Context, job domain.Job) error
	dequeueFn func(ctx context.Context) (*domain.Job, error)
	jobs      []domain.Job
	calls     int
}

var _ domain.Queue = (*mockWebhookSchedulerQueue)(nil)

func (m *mockWebhookSchedulerQueue) Enqueue(ctx context.Context, job domain.Job) error {
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

func (m *mockWebhookSchedulerQueue) Dequeue(ctx context.Context) (*domain.Job, error) {
	if m.dequeueFn != nil {
		return m.dequeueFn(ctx)
	}
	return nil, errors.New("not implemented")
}

// ── Task 11.1: NewWebhookScheduler construction ───────────────────────────────

func TestNewWebhookScheduler_Construction(t *testing.T) {
	repo := &mockSchedulerEndpointRepo{}
	secret := &mockSchedulerSecretReader{}
	q := &mockWebhookSchedulerQueue{}

	s := NewWebhookScheduler(repo, secret, q)

	if s == nil {
		t.Fatal("expected non-nil scheduler")
		return
	}
	if s.endpointRepo == nil {
		t.Fatal("expected endpointRepo to be set")
	}
	if s.secretReader == nil {
		t.Fatal("expected secretReader to be set")
	}
	if s.queue == nil {
		t.Fatal("expected queue to be set")
	}
}

// ── Task 11.2: Dispatch with zero active endpoints enqueues no jobs ──────────

func TestWebhookScheduler_Dispatch_ZeroEndpoints_NoJobsEnqueued(t *testing.T) {
	orgID := uuid.New()
	q := &mockWebhookSchedulerQueue{}

	s := NewWebhookScheduler(
		&mockSchedulerEndpointRepo{
			listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
				return []domain.WebhookEndpoint{}, nil
			},
		},
		&mockSchedulerSecretReader{
			getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "valid-secret", nil
			},
		},
		q,
	)

	err := s.Dispatch(context.Background(), orgID, "customer.created", "production", []byte(`{"id":"1"}`))
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

func TestWebhookScheduler_Dispatch_NEndpoints_EnqueuesNJobs(t *testing.T) {
	orgID := uuid.New()
	endpoints := []domain.WebhookEndpoint{
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook1"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook2"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook3"},
	}

	q := &mockWebhookSchedulerQueue{}

	s := NewWebhookScheduler(
		&mockSchedulerEndpointRepo{
			listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
				return endpoints, nil
			},
		},
		&mockSchedulerSecretReader{
			getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "test-secret", nil
			},
		},
		q,
	)

	err := s.Dispatch(context.Background(), orgID, "customer.created", "production", []byte(`{"id":"1"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	q.mu.Lock()
	defer q.mu.Unlock()

	if len(q.jobs) != 3 {
		t.Fatalf("expected 3 enqueued jobs, got %d", len(q.jobs))
	}

	for i, job := range q.jobs {
		if job.Type != domain.JobTypeWebhook {
			t.Errorf("job[%d]: expected Type=%q, got %q", i, domain.JobTypeWebhook, job.Type)
		}
		if job.AttemptNumber != 1 {
			t.Errorf("job[%d]: expected AttemptNumber=1, got %d", i, job.AttemptNumber)
		}
		if job.ID == uuid.Nil {
			t.Errorf("job[%d]: expected non-zero ID", i)
		}

		var wj domain.WebhookJob
		if err := json.Unmarshal(job.Payload, &wj); err != nil {
			t.Errorf("job[%d]: payload is not valid WebhookJob JSON: %v", i, err)
			continue
		}
		if wj.Secret != "test-secret" {
			t.Errorf("job[%d]: expected Secret='test-secret', got %q", i, wj.Secret)
		}
		if wj.EndpointURL != endpoints[i].URL {
			t.Errorf("job[%d]: expected EndpointURL=%q, got %q", i, endpoints[i].URL, wj.EndpointURL)
		}
		if wj.EndpointID != endpoints[i].ID {
			t.Errorf("job[%d]: expected EndpointID=%s, got %s", i, endpoints[i].ID, wj.EndpointID)
		}
	}
}

// ── Task 11.4: Dispatch continues enqueuing when Enqueue fails for a subset ──

func TestWebhookScheduler_Dispatch_EnqueuePartialFailure_ContinuesAll(t *testing.T) {
	orgID := uuid.New()
	endpoints := []domain.WebhookEndpoint{
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook1"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook2"},
		{ID: uuid.New(), OrganizationID: orgID, URL: "https://example.com/hook3"},
	}

	callCount := 0
	q := &mockWebhookSchedulerQueue{
		enqueueFn: func(_ context.Context, _ domain.Job) error {
			callCount++
			if callCount == 2 {
				return errors.New("queue temporarily unavailable")
			}
			return nil
		},
	}

	s := NewWebhookScheduler(
		&mockSchedulerEndpointRepo{
			listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
				return endpoints, nil
			},
		},
		&mockSchedulerSecretReader{
			getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
				return "test-secret", nil
			},
		},
		q,
	)

	err := s.Dispatch(context.Background(), orgID, "customer.created", "production", []byte(`{"id":"1"}`))
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
