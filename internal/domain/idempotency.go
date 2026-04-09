package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// IdempotencyRecord holds the cached response for an idempotent request.
type IdempotencyRecord struct {
	ResponseCode int    `json:"response_code"`
	ResponseBody []byte `json:"response_body"`
}

// IdempotencyStorage abstracts idempotency key persistence so the backing
// store (Postgres, Redis, …) can be swapped without touching callers.
type IdempotencyStorage interface {
	// Check verifies whether a key already exists for the given organization.
	// Returns the record when found, nil when missing.
	Check(ctx context.Context, key string, orgID uuid.UUID) (*IdempotencyRecord, error)

	// Save persists the operation result bound to key with the given TTL.
	Save(ctx context.Context, key string, orgID uuid.UUID, requestPath string, record IdempotencyRecord, ttl time.Duration) error
}
