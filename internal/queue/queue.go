package queue

import "i9pay/internal/domain"

// New returns a domain.Queue implementation based on the given backend name.
// Currently only "memory" is supported. Returns nil for unrecognized backends.
func New(backend string, bufferSize int) domain.Queue {
	switch backend {
	case "memory":
		return NewInMemoryQueue(bufferSize)
	default:
		return nil
	}
}
