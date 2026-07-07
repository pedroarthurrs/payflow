package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// genWebhookJob generates an arbitrary domain.WebhookJob for property tests.
func genWebhookJob(t *rapid.T) domain.WebhookJob {
	return domain.WebhookJob{
		OrganizationID: uuid.New(),
		EndpointID:     uuid.New(),
		EndpointURL:    "http://example.com/webhook",
		Secret:         rapid.String().Draw(t, "secret"),
		EventType:      rapid.String().Draw(t, "event_type"),
		Environment:    rapid.String().Draw(t, "environment"),
		Payload:        rapid.SliceOf(rapid.Byte()).Draw(t, "payload"),
	}
}

// genNonWebhookJobType generates a JobType that is NOT "webhook".
func genNonWebhookJobType(t *rapid.T) domain.JobType {
	// Generate a string that is not "webhook"
	s := rapid.StringMatching(`[a-z]{1,10}`).Draw(t, "job_type")
	if s == "webhook" {
		s = "email"
	}
	return domain.JobType(s)
}

// Feature: generic-queue, Property 7: Unknown job type is discarded by Worker
func TestProperty_Worker_UnknownTypeDiscarded(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		jobType := genNonWebhookJobType(t)

		var httpCalls int
		var mu sync.Mutex

		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			httpCalls++
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
			_, _ = fmt.Fprint(w, `{"ok":true}`)
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

		// Build a job with a non-webhook type but valid-looking payload
		wj := domain.WebhookJob{
			OrganizationID: uuid.New(),
			EndpointID:     uuid.New(),
			EndpointURL:    srv.URL,
			Secret:         "secret",
			EventType:      "test.event",
			Environment:    "test",
			Payload:        []byte(`{"data":"test"}`),
		}
		payload, _ := json.Marshal(wj)

		job := domain.Job{
			ID:            uuid.New(),
			Type:          jobType,
			Payload:       payload,
			ScheduledAt:   time.Now(),
			AttemptNumber: 1,
		}

		ctx := context.Background()
		w.process(ctx, &job)

		mu.Lock()
		defer mu.Unlock()

		if httpCalls != 0 {
			t.Fatalf("expected zero HTTP calls for job type %q, got %d", jobType, httpCalls)
		}
		if enqueueCount != 0 {
			t.Fatalf("expected zero enqueued jobs for job type %q, got %d", jobType, enqueueCount)
		}
	})
}

// Feature: generic-queue, Property 8: Retry preserves Job identity and increments AttemptNumber
func TestProperty_Worker_RetryPreservesIdentity(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// AttemptNumber in range 1..4 so retry will happen (not dead letter)
		attemptNumber := rapid.Int32Range(1, 4).Draw(t, "attempt_number")

		// Generate a valid WebhookJob payload
		wj := genWebhookJob(t)

		// Use a real HTTP server that always returns 500
		srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprint(w, "error")
		}))
		defer srv.Close()

		// Override the endpoint URL to point to our test server
		wj.EndpointURL = srv.URL

		payload, err := json.Marshal(wj)
		if err != nil {
			t.Fatalf("failed to marshal WebhookJob: %v", err)
		}

		originalID := uuid.New()
		originalJob := domain.Job{
			ID:            originalID,
			Type:          domain.JobTypeWebhook,
			Payload:       payload,
			ScheduledAt:   time.Now(),
			AttemptNumber: attemptNumber,
		}

		var enqueuedJobs []domain.Job
		var mu sync.Mutex

		mq := &mockQueue{
			enqueueFn: func(ctx context.Context, job domain.Job) error {
				mu.Lock()
				enqueuedJobs = append(enqueuedJobs, job)
				mu.Unlock()
				return nil
			},
		}

		logRepo := &mockWebhookLogRepo{
			createFn: func(ctx context.Context, input domain.CreateWebhookLogInput) error {
				return nil
			},
		}

		w := NewWebhookWorker(mq, logRepo, srv.Client(), 1)

		beforeEnqueue := time.Now()
		ctx := context.Background()
		w.process(ctx, &originalJob)

		mu.Lock()
		defer mu.Unlock()

		if len(enqueuedJobs) != 1 {
			t.Fatalf("expected exactly 1 re-enqueued job, got %d", len(enqueuedJobs))
		}

		requeued := enqueuedJobs[0]

		// Same ID
		if requeued.ID != originalID {
			t.Fatalf("expected same ID %v, got %v", originalID, requeued.ID)
		}

		// Same Type
		if requeued.Type != domain.JobTypeWebhook {
			t.Fatalf("expected Type=%q, got %q", domain.JobTypeWebhook, requeued.Type)
		}

		// Same Payload
		if !bytes.Equal(requeued.Payload, payload) {
			t.Fatalf("expected same Payload, got different")
		}

		// AttemptNumber incremented by 1
		if requeued.AttemptNumber != attemptNumber+1 {
			t.Fatalf("expected AttemptNumber=%d, got %d", attemptNumber+1, requeued.AttemptNumber)
		}

		// ScheduledAt is in the future (after the time we started)
		if !requeued.ScheduledAt.After(beforeEnqueue) {
			t.Fatalf("expected ScheduledAt in the future, got %v (before %v)", requeued.ScheduledAt, beforeEnqueue)
		}
	})
}
