package queue

import "context"

// InMemoryQueue implementa Queue usando um canal Go com buffer.
// Adequada para desenvolvimento e testes. Jobs são perdidos em caso de restart.
type InMemoryQueue struct {
	ch chan WebhookJob
}

// NewInMemoryQueue cria uma InMemoryQueue com o tamanho de buffer especificado.
func NewInMemoryQueue(bufferSize int) *InMemoryQueue {
	return &InMemoryQueue{ch: make(chan WebhookJob, bufferSize)}
}

// Enqueue adds a job to the queue. Blocks if the buffer is full; returns ctx.Err() if the context is canceled.
func (q *InMemoryQueue) Enqueue(ctx context.Context, job WebhookJob) error {
	select {
	case q.ch <- job:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// Dequeue removes and returns the next job from the queue. Blocks until a job is available or the context is canceled.
func (q *InMemoryQueue) Dequeue(ctx context.Context) (*WebhookJob, error) {
	select {
	case job := <-q.ch:
		return &job, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}
