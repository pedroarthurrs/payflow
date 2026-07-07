package queue

import (
	"context"

	"payflow/internal/domain"
)

// InMemoryQueue implements domain.Queue using a buffered Go channel.
// Suitable for development and testing. Jobs are lost on restart.
type InMemoryQueue struct {
	ch chan domain.Job
}

// NewInMemoryQueue creates an InMemoryQueue with the given buffer size.
func NewInMemoryQueue(bufferSize int) *InMemoryQueue {
	return &InMemoryQueue{ch: make(chan domain.Job, bufferSize)}
}

// Enqueue adds a job to the queue. Blocks if the buffer is full; returns ctx.Err() if the context is canceled.
func (q *InMemoryQueue) Enqueue(ctx context.Context, job domain.Job) error {
	select {
	case q.ch <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue removes and returns the next job from the queue. Blocks until a job is available or the context is canceled.
func (q *InMemoryQueue) Dequeue(ctx context.Context) (*domain.Job, error) {
	select {
	case job := <-q.ch:
		return &job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

var _ domain.Queue = (*InMemoryQueue)(nil)
