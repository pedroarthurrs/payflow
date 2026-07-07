package domain

import (
	"context"
	"time"
)

// CleanupRepository defines operations for periodic data purging.
type CleanupRepository interface {
	// DeleteCredentialLogsOlderThan removes credential logs created before the given cutoff.
	// Returns the number of rows deleted.
	DeleteCredentialLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)

	// DeleteWebhookLogsOlderThan removes webhook logs created before the given cutoff.
	// Returns the number of rows deleted.
	DeleteWebhookLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error)

	// DeleteExpiredIdempotencyKeys removes idempotency keys whose expires_at <= NOW().
	// Returns the number of rows deleted.
	DeleteExpiredIdempotencyKeys(ctx context.Context) (int64, error)
}
