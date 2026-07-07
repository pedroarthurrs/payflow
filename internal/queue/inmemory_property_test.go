package queue

import (
	"bytes"
	"context"
	"testing"
	"time"
	"unicode"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

func genJob(t *rapid.T) domain.Job {
	return domain.Job{
		ID:            uuid.New(),
		Type:          domain.JobType(rapid.StringOf(rapid.RuneFrom(nil, unicode.Letter)).Draw(t, "type")),
		Payload:       rapid.SliceOf(rapid.Byte()).Draw(t, "payload"),
		ScheduledAt:   time.Now().Add(time.Duration(rapid.Int64Range(-1e9, 1e9).Draw(t, "sched")) * time.Nanosecond),
		AttemptNumber: rapid.Int32Range(1, 10).Draw(t, "attempt"),
	}
}

// Feature: generic-queue, Property 1: Queue round-trip
func TestProperty_InMemoryQueue_RoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		job := rapid.Custom(genJob).Draw(t, "job")

		q := NewInMemoryQueue(1)
		ctx := context.Background()

		if err := q.Enqueue(ctx, job); err != nil {
			t.Fatalf("Enqueue failed: %v", err)
		}

		got, err := q.Dequeue(ctx)
		if err != nil {
			t.Fatalf("Dequeue failed: %v", err)
		}

		if got.ID != job.ID {
			t.Errorf("ID mismatch: got %v, want %v", got.ID, job.ID)
		}
		if got.Type != job.Type {
			t.Errorf("Type mismatch: got %q, want %q", got.Type, job.Type)
		}
		if !bytes.Equal(got.Payload, job.Payload) {
			t.Errorf("Payload mismatch: got %v, want %v", got.Payload, job.Payload)
		}
		if !got.ScheduledAt.Equal(job.ScheduledAt) {
			t.Errorf("ScheduledAt mismatch: got %v, want %v", got.ScheduledAt, job.ScheduledAt)
		}
		if got.AttemptNumber != job.AttemptNumber {
			t.Errorf("AttemptNumber mismatch: got %d, want %d", got.AttemptNumber, job.AttemptNumber)
		}
	})
}

// Feature: generic-queue, Property 2: Queue FIFO ordering
func TestProperty_InMemoryQueue_FIFO(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		jobs := rapid.SliceOf(rapid.Custom(genJob)).Draw(t, "jobs")
		if len(jobs) == 0 {
			return
		}

		q := NewInMemoryQueue(len(jobs))
		ctx := context.Background()

		for _, job := range jobs {
			if err := q.Enqueue(ctx, job); err != nil {
				t.Fatalf("Enqueue failed: %v", err)
			}
		}

		for i, want := range jobs {
			got, err := q.Dequeue(ctx)
			if err != nil {
				t.Fatalf("Dequeue[%d] failed: %v", i, err)
			}
			if got.ID != want.ID {
				t.Errorf("FIFO violation at index %d: got ID %v, want %v", i, got.ID, want.ID)
			}
		}
	})
}

// Feature: generic-queue, Property 3: Full queue returns error on Enqueue
func TestProperty_InMemoryQueue_FullQueueError(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		size := int(rapid.IntRange(1, 10).Draw(t, "size"))
		q := NewInMemoryQueue(size)

		ctx := context.Background()
		job := rapid.Custom(genJob).Draw(t, "job")

		// Fill the queue
		for i := 0; i < size; i++ {
			if err := q.Enqueue(ctx, job); err != nil {
				t.Fatalf("Enqueue[%d] failed unexpectedly: %v", i, err)
			}
		}

		// Cancel context and try to enqueue one more
		canceledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := q.Enqueue(canceledCtx, job)
		if err == nil {
			t.Error("expected non-nil error when enqueuing into full queue with canceled context, got nil")
		}
	})
}

// Feature: generic-queue, Property 4: Canceled context returns ctx.Err()
func TestProperty_InMemoryQueue_CanceledContext(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		q := NewInMemoryQueue(10)
		job := rapid.Custom(genJob).Draw(t, "job")

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		// Test Enqueue with canceled context on a full queue
		// Fill the queue first so Enqueue would block
		fillCtx := context.Background()
		for i := 0; i < 10; i++ {
			_ = q.Enqueue(fillCtx, job)
		}

		enqErr := q.Enqueue(ctx, job)
		if enqErr != ctx.Err() {
			t.Errorf("Enqueue: got error %v, want ctx.Err() = %v", enqErr, ctx.Err())
		}

		// Test Dequeue with canceled context on an empty queue
		emptyQ := NewInMemoryQueue(10)
		_, deqErr := emptyQ.Dequeue(ctx)
		if deqErr != ctx.Err() {
			t.Errorf("Dequeue: got error %v, want ctx.Err() = %v", deqErr, ctx.Err())
		}
	})
}
