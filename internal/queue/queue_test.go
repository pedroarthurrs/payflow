package queue

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestInMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewInMemoryQueue(10)

	want := WebhookJob{
		OrganizationID: uuid.New(),
		EndpointID:     uuid.New(),
		EndpointURL:    "https://example.com/webhook",
		Secret:         "test-secret-key",
		EventType:      "payment.completed",
		Environment:    "production",
		Payload:        []byte(`{"amount":1000,"currency":"BRL"}`),
		AttemptNumber:  1,
		ScheduledAt:    time.Now().Truncate(time.Millisecond),
	}

	ctx := context.Background()

	if err := q.Enqueue(ctx, want); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	got, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}

	if got.OrganizationID != want.OrganizationID {
		t.Errorf("OrganizationID = %v, want %v", got.OrganizationID, want.OrganizationID)
	}
	if got.EndpointID != want.EndpointID {
		t.Errorf("EndpointID = %v, want %v", got.EndpointID, want.EndpointID)
	}
	if got.EndpointURL != want.EndpointURL {
		t.Errorf("EndpointURL = %q, want %q", got.EndpointURL, want.EndpointURL)
	}
	if got.Secret != want.Secret {
		t.Errorf("Secret = %q, want %q", got.Secret, want.Secret)
	}
	if got.EventType != want.EventType {
		t.Errorf("EventType = %q, want %q", got.EventType, want.EventType)
	}
	if got.Environment != want.Environment {
		t.Errorf("Environment = %q, want %q", got.Environment, want.Environment)
	}
	if !bytes.Equal(got.Payload, want.Payload) {
		t.Errorf("Payload = %q, want %q", got.Payload, want.Payload)
	}
	if got.AttemptNumber != want.AttemptNumber {
		t.Errorf("AttemptNumber = %d, want %d", got.AttemptNumber, want.AttemptNumber)
	}
	if !got.ScheduledAt.Equal(want.ScheduledAt) {
		t.Errorf("ScheduledAt = %v, want %v", got.ScheduledAt, want.ScheduledAt)
	}
}

func TestInMemoryQueue_DequeueEmptyCancelled(t *testing.T) {
	q := NewInMemoryQueue(10)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := q.Dequeue(ctx)
	if got != nil {
		t.Errorf("Dequeue returned non-nil job: %+v", got)
	}
	if err != context.Canceled {
		t.Errorf("Dequeue error = %v, want %v", err, context.Canceled)
	}
}
