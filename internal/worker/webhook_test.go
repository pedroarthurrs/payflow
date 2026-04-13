package worker

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// --- Mock types ---

type mockQueue struct {
	enqueueFn func(ctx context.Context, job domain.Job) error
	dequeueFn func(ctx context.Context) (*domain.Job, error)
}

func (m *mockQueue) Enqueue(ctx context.Context, job domain.Job) error {
	return m.enqueueFn(ctx, job)
}

func (m *mockQueue) Dequeue(ctx context.Context) (*domain.Job, error) {
	return m.dequeueFn(ctx)
}

type mockWebhookLogRepo struct {
	createFn func(ctx context.Context, input domain.CreateWebhookLogInput) error
}

func (m *mockWebhookLogRepo) Create(ctx context.Context, input domain.CreateWebhookLogInput) error {
	return m.createFn(ctx, input)
}

func (m *mockWebhookLogRepo) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.WebhookLog, error) {
	return nil, nil
}

func (m *mockWebhookLogRepo) List(ctx context.Context, orgID uuid.UUID, input domain.ListWebhookLogsInput) ([]domain.WebhookLog, error) {
	return nil, nil
}

// makeWebhookJob builds a domain.Job wrapping a WebhookJob payload.
func makeWebhookJob(t *testing.T, wj domain.WebhookJob, attemptNumber int32) domain.Job {
	t.Helper()
	payload, err := json.Marshal(wj)
	if err != nil {
		t.Fatalf("failed to marshal WebhookJob: %v", err)
	}
	return domain.Job{
		ID:            uuid.New(),
		Type:          domain.JobTypeWebhook,
		Payload:       payload,
		ScheduledAt:   time.Now(),
		AttemptNumber: attemptNumber,
	}
}

// --- Test 10.1: deliver with HTTP 200 success ---

func TestDeliver_Success200(t *testing.T) {
	secret := "my-test-secret"
	eventType := "customer.created"
	payload := []byte(`{"id":"123","name":"Test Customer"}`)

	var capturedReq struct {
		mu          sync.Mutex
		contentType string
		event       string
		signature   string
		body        []byte
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq.mu.Lock()
		defer capturedReq.mu.Unlock()
		capturedReq.contentType = r.Header.Get("Content-Type")
		capturedReq.event = r.Header.Get("X-Webhook-Event")
		capturedReq.signature = r.Header.Get("X-Webhook-Signature")
		capturedReq.body, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
		_, _ = fmt.Fprint(w, `{"ok":true}`)
	}))
	defer srv.Close()

	w := NewWebhookWorker(nil, nil, srv.Client(), 1)

	wj := domain.WebhookJob{
		OrganizationID: uuid.New(),
		EndpointID:     uuid.New(),
		EndpointURL:    srv.URL,
		Secret:         secret,
		EventType:      eventType,
		Environment:    "test",
		Payload:        payload,
	}

	ctx := context.Background()
	result := w.deliver(ctx, &wj, 1)

	// Assert success
	if !result.Success {
		t.Errorf("expected Success=true, got false")
	}
	if result.ResponseStatus == nil || *result.ResponseStatus != 200 {
		t.Errorf("expected ResponseStatus=200, got %v", result.ResponseStatus)
	}
	if result.DurationMs == nil || *result.DurationMs < 0 {
		t.Errorf("expected DurationMs >= 0, got %v", result.DurationMs)
	}

	// Verify captured headers
	capturedReq.mu.Lock()
	defer capturedReq.mu.Unlock()

	if capturedReq.contentType != "application/json" {
		t.Errorf("Content-Type = %q, want %q", capturedReq.contentType, "application/json")
	}
	if capturedReq.event != eventType {
		t.Errorf("X-Webhook-Event = %q, want %q", capturedReq.event, eventType)
	}

	// Verify HMAC-SHA256 signature
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(payload)
	expectedSig := hex.EncodeToString(mac.Sum(nil))
	if capturedReq.signature != expectedSig {
		t.Errorf("X-Webhook-Signature = %q, want %q", capturedReq.signature, expectedSig)
	}

	// Verify body was sent correctly
	if !bytes.Equal(capturedReq.body, payload) {
		t.Errorf("request body = %q, want %q", capturedReq.body, payload)
	}
}

// --- Test 10.2: deliver with HTTP 500 failure ---

func TestDeliver_Failure500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "internal server error")
	}))
	defer srv.Close()

	w := NewWebhookWorker(nil, nil, srv.Client(), 1)

	wj := domain.WebhookJob{
		OrganizationID: uuid.New(),
		EndpointID:     uuid.New(),
		EndpointURL:    srv.URL,
		Secret:         "secret",
		EventType:      "invoice.paid",
		Environment:    "production",
		Payload:        []byte(`{"invoice_id":"456"}`),
	}

	ctx := context.Background()
	result := w.deliver(ctx, &wj, 1)

	if result.Success {
		t.Errorf("expected Success=false, got true")
	}
	if result.ResponseStatus == nil || *result.ResponseStatus != 500 {
		t.Errorf("expected ResponseStatus=500, got %v", result.ResponseStatus)
	}
	if result.ResponseBody == nil || *result.ResponseBody != "internal server error" {
		t.Errorf("expected ResponseBody to contain 'internal server error', got %v", result.ResponseBody)
	}
}

// --- Test 10.3: process with AttemptNumber=1 and failed delivery (re-enqueue) ---

func TestProcess_Attempt1_FailedDelivery_Reenqueues(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "error")
	}))
	defer srv.Close()

	var enqueuedJobs []domain.Job
	var mu sync.Mutex

	mq := &mockQueue{
		enqueueFn: func(ctx context.Context, job domain.Job) error {
			mu.Lock()
			defer mu.Unlock()
			enqueuedJobs = append(enqueuedJobs, job)
			return nil
		},
	}

	logRepo := &mockWebhookLogRepo{
		createFn: func(ctx context.Context, input domain.CreateWebhookLogInput) error {
			return nil
		},
	}

	w := NewWebhookWorker(mq, logRepo, srv.Client(), 1)

	wj := domain.WebhookJob{
		OrganizationID: uuid.New(),
		EndpointID:     uuid.New(),
		EndpointURL:    srv.URL,
		Secret:         "secret",
		EventType:      "customer.created",
		Environment:    "test",
		Payload:        []byte(`{"data":"test"}`),
	}
	job := makeWebhookJob(t, wj, 1)

	ctx := context.Background()
	now := time.Now()
	w.process(ctx, &job)

	mu.Lock()
	defer mu.Unlock()

	if len(enqueuedJobs) != 1 {
		t.Fatalf("expected exactly 1 enqueued job, got %d", len(enqueuedJobs))
	}

	requeued := enqueuedJobs[0]
	if requeued.AttemptNumber != 2 {
		t.Errorf("expected AttemptNumber=2, got %d", requeued.AttemptNumber)
	}
	if requeued.ID != job.ID {
		t.Errorf("expected same ID %v, got %v", job.ID, requeued.ID)
	}
	if !bytes.Equal(requeued.Payload, job.Payload) {
		t.Errorf("expected same Payload, got different")
	}

	expectedScheduledAt := now.Add(1 * time.Minute)
	tolerance := 5 * time.Second
	diff := requeued.ScheduledAt.Sub(expectedScheduledAt)
	if diff < -tolerance || diff > tolerance {
		t.Errorf("ScheduledAt = %v, expected approximately %v (diff: %v)", requeued.ScheduledAt, expectedScheduledAt, diff)
	}
}

// --- Test 10.4: process with AttemptNumber=5 and failed delivery (DeadLetter) ---

func TestProcess_Attempt5_FailedDelivery_DeadLetter(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "error")
	}))
	defer srv.Close()

	var enqueueCount int
	var mu sync.Mutex

	mq := &mockQueue{
		enqueueFn: func(ctx context.Context, job domain.Job) error {
			mu.Lock()
			defer mu.Unlock()
			enqueueCount++
			return nil
		},
	}

	logRepo := &mockWebhookLogRepo{
		createFn: func(ctx context.Context, input domain.CreateWebhookLogInput) error {
			return nil
		},
	}

	w := NewWebhookWorker(mq, logRepo, srv.Client(), 1)

	wj := domain.WebhookJob{
		OrganizationID: uuid.New(),
		EndpointID:     uuid.New(),
		EndpointURL:    srv.URL,
		Secret:         "secret",
		EventType:      "customer.deleted",
		Environment:    "production",
		Payload:        []byte(`{"data":"test"}`),
	}
	job := makeWebhookJob(t, wj, 5)

	ctx := context.Background()
	w.process(ctx, &job)

	mu.Lock()
	defer mu.Unlock()

	if enqueueCount != 0 {
		t.Errorf("expected zero enqueued jobs (dead letter), got %d", enqueueCount)
	}
}

// --- Test: process with unknown job type (discarded) ---

func TestProcess_UnknownJobType_Discarded(t *testing.T) {
	var httpCalls int
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		httpCalls++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var enqueueCount int
	mq := &mockQueue{
		enqueueFn: func(ctx context.Context, job domain.Job) error {
			mu.Lock()
			enqueueCount++
			mu.Unlock()
			return nil
		},
	}

	w := NewWebhookWorker(mq, nil, srv.Client(), 1)

	job := domain.Job{
		ID:            uuid.New(),
		Type:          domain.JobType("email"),
		Payload:       []byte(`{"to":"test@example.com"}`),
		ScheduledAt:   time.Now(),
		AttemptNumber: 1,
	}

	ctx := context.Background()
	w.process(ctx, &job)

	mu.Lock()
	defer mu.Unlock()

	if httpCalls != 0 {
		t.Errorf("expected zero HTTP calls for unknown job type, got %d", httpCalls)
	}
	if enqueueCount != 0 {
		t.Errorf("expected zero enqueued jobs for unknown job type, got %d", enqueueCount)
	}
}

// --- Test: process with invalid payload (discarded) ---

func TestProcess_InvalidPayload_Discarded(t *testing.T) {
	var httpCalls int
	var mu sync.Mutex

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		httpCalls++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	var enqueueCount int
	mq := &mockQueue{
		enqueueFn: func(ctx context.Context, job domain.Job) error {
			mu.Lock()
			enqueueCount++
			mu.Unlock()
			return nil
		},
	}

	w := NewWebhookWorker(mq, nil, srv.Client(), 1)

	job := domain.Job{
		ID:            uuid.New(),
		Type:          domain.JobTypeWebhook,
		Payload:       []byte("not-json"),
		ScheduledAt:   time.Now(),
		AttemptNumber: 1,
	}

	ctx := context.Background()
	w.process(ctx, &job)

	mu.Lock()
	defer mu.Unlock()

	if httpCalls != 0 {
		t.Errorf("expected zero HTTP calls for invalid payload, got %d", httpCalls)
	}
	if enqueueCount != 0 {
		t.Errorf("expected zero enqueued jobs for invalid payload, got %d", enqueueCount)
	}
}

// --- Test 10.5: Start with context cancellation (no goroutine leak) ---

func TestStart_ContextCancellation(t *testing.T) {
	mq := &mockQueue{
		dequeueFn: func(ctx context.Context) (*domain.Job, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}

	w := NewWebhookWorker(mq, nil, &http.Client{}, 3)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan struct{})
	go func() {
		w.Start(ctx)
		close(done)
	}()

	// Give workers time to start and block on Dequeue
	time.Sleep(50 * time.Millisecond)
	cancel()

	select {
	case <-done:
		// Start returned — all goroutines cleaned up
	case <-time.After(2 * time.Second):
		t.Fatal("Start did not return within 2 seconds after context cancellation — possible goroutine leak")
	}
}
