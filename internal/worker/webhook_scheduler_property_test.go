package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// captureQueue is a mock queue that captures all enqueued jobs.
type captureQueue struct {
	jobs []domain.Job
}

func (q *captureQueue) Enqueue(_ context.Context, job domain.Job) error {
	q.jobs = append(q.jobs, job)
	return nil
}

func (q *captureQueue) Dequeue(_ context.Context) (*domain.Job, error) {
	return nil, nil
}

// Feature: generic-queue, Property 5: WebhookJob serialization round-trip
func TestProperty_Scheduler_WebhookJobRoundTrip(t *testing.T) {
	// Validates: Requirements 4.1, 5.3
	rapid.Check(t, func(t *rapid.T) {
		original := genWebhookJob(t)

		cq := &captureQueue{}
		orgID := original.OrganizationID

		s := NewWebhookScheduler(
			&mockSchedulerEndpointRepo{
				listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
					return []domain.WebhookEndpoint{
						{
							ID:             original.EndpointID,
							OrganizationID: orgID,
							URL:            original.EndpointURL,
						},
					}, nil
				},
			},
			&mockSchedulerSecretReader{
				getWebhookSecretFn: func(_ context.Context, _ uuid.UUID) (string, error) {
					return original.Secret, nil
				},
			},
			cq,
		)

		err := s.Dispatch(context.Background(), orgID, original.EventType, original.Environment, original.Payload)
		if err != nil {
			t.Fatalf("Dispatch returned unexpected error: %v", err)
		}

		if len(cq.jobs) != 1 {
			t.Fatalf("expected 1 enqueued job, got %d", len(cq.jobs))
		}

		var decoded domain.WebhookJob
		if err := json.Unmarshal(cq.jobs[0].Payload, &decoded); err != nil {
			t.Fatalf("failed to unmarshal Job.Payload back to WebhookJob: %v", err)
		}

		if decoded.OrganizationID != original.OrganizationID {
			t.Errorf("OrganizationID mismatch: got %v, want %v", decoded.OrganizationID, original.OrganizationID)
		}
		if decoded.EndpointID != original.EndpointID {
			t.Errorf("EndpointID mismatch: got %v, want %v", decoded.EndpointID, original.EndpointID)
		}
		if decoded.EndpointURL != original.EndpointURL {
			t.Errorf("EndpointURL mismatch: got %q, want %q", decoded.EndpointURL, original.EndpointURL)
		}
		if decoded.Secret != original.Secret {
			t.Errorf("Secret mismatch: got %q, want %q", decoded.Secret, original.Secret)
		}
		if decoded.EventType != original.EventType {
			t.Errorf("EventType mismatch: got %q, want %q", decoded.EventType, original.EventType)
		}
		if decoded.Environment != original.Environment {
			t.Errorf("Environment mismatch: got %q, want %q", decoded.Environment, original.Environment)
		}
		if !bytes.Equal(decoded.Payload, original.Payload) {
			t.Errorf("Payload mismatch: got %v, want %v", decoded.Payload, original.Payload)
		}
	})
}

// Feature: generic-queue, Property 6: Dispatched Job fields invariant
func TestProperty_Scheduler_JobFieldsInvariant(t *testing.T) {
	// Validates: Requirements 4.2
	rapid.Check(t, func(t *rapid.T) {
		endpointURL := rapid.StringMatching(`https://[a-z]{3,8}\.[a-z]{2,4}/[a-z]{0,8}`).Draw(t, "endpoint_url")
		orgID := uuid.New()
		endpointID := uuid.New()

		cq := &captureQueue{}

		s := NewWebhookScheduler(
			&mockSchedulerEndpointRepo{
				listActiveByEventFn: func(_ context.Context, _ uuid.UUID, _, _ string) ([]domain.WebhookEndpoint, error) {
					return []domain.WebhookEndpoint{
						{ID: endpointID, OrganizationID: orgID, URL: endpointURL},
					}, nil
				},
			},
			&mockSchedulerSecretReader{},
			cq,
		)

		eventType := rapid.String().Draw(t, "event_type")
		environment := rapid.String().Draw(t, "environment")
		payload := rapid.SliceOf(rapid.Byte()).Draw(t, "payload")

		err := s.Dispatch(context.Background(), orgID, eventType, environment, payload)
		if err != nil {
			t.Fatalf("Dispatch returned unexpected error: %v", err)
		}

		if len(cq.jobs) != 1 {
			t.Fatalf("expected 1 enqueued job, got %d", len(cq.jobs))
		}

		job := cq.jobs[0]

		if job.Type != domain.JobTypeWebhook {
			t.Errorf("expected JobType=%q, got %q", domain.JobTypeWebhook, job.Type)
		}
		if job.ID == uuid.Nil {
			t.Error("expected non-zero job ID")
		}
		if job.AttemptNumber != 1 {
			t.Errorf("expected AttemptNumber=1, got %d", job.AttemptNumber)
		}
		if job.ScheduledAt.IsZero() {
			t.Error("expected non-zero ScheduledAt")
		}
	})
}
