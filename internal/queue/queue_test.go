package queue

import (
	"bytes"
	"context"
	"testing"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

func TestInMemoryQueue_EnqueueDequeue(t *testing.T) {
	q := NewInMemoryQueue(10)

	want := domain.Job{
		ID:            uuid.New(),
		Type:          domain.JobTypeWebhook,
		Payload:       []byte(`{"amount":1000,"currency":"BRL"}`),
		ScheduledAt:   time.Now().Truncate(time.Millisecond),
		AttemptNumber: 1,
	}

	ctx := context.Background()

	if err := q.Enqueue(ctx, want); err != nil {
		t.Fatalf("Enqueue failed: %v", err)
	}

	got, err := q.Dequeue(ctx)
	if err != nil {
		t.Fatalf("Dequeue failed: %v", err)
	}

	if got.ID != want.ID {
		t.Errorf("ID = %v, want %v", got.ID, want.ID)
	}
	if got.Type != want.Type {
		t.Errorf("Type = %q, want %q", got.Type, want.Type)
	}
	if !bytes.Equal(got.Payload, want.Payload) {
		t.Errorf("Payload = %q, want %q", got.Payload, want.Payload)
	}
	if !got.ScheduledAt.Equal(want.ScheduledAt) {
		t.Errorf("ScheduledAt = %v, want %v", got.ScheduledAt, want.ScheduledAt)
	}
	if got.AttemptNumber != want.AttemptNumber {
		t.Errorf("AttemptNumber = %d, want %d", got.AttemptNumber, want.AttemptNumber)
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
