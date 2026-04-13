package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// JobType identifies the category of a queued job (e.g. "webhook", "email").
type JobType string

// JobTypeWebhook is the job type for webhook delivery jobs.
const JobTypeWebhook JobType = "webhook"

// Job is the generic unit of work stored in the queue.
// The job-specific data is serialized as JSON by the producer and stored in Payload.
type Job struct {
	ID            uuid.UUID
	Type          JobType
	Payload       []byte
	ScheduledAt   time.Time
	AttemptNumber int32
}

// Queue abstracts a generic message queue.
// Concrete implementations may use InMemoryQueue, SQSQueue, Redis, etc.
type Queue interface {
	// Enqueue adds a job to the queue. Returns an error if the queue is full or unavailable.
	Enqueue(ctx context.Context, job Job) error
	// Dequeue removes and returns the next job from the queue. Blocks until a job is
	// available or the context is canceled, returning nil, ctx.Err() on cancellation.
	Dequeue(ctx context.Context) (*Job, error)
}
