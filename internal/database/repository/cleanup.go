package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	credentiallogdb "i9pay/internal/database/dbgen/credentiallog"
	idempotencydb "i9pay/internal/database/dbgen/idempotency"
	webhooklogdb "i9pay/internal/database/dbgen/webhooklog"
	"i9pay/internal/domain"
)

// CleanupRepo implements domain.CleanupRepository backed by PostgreSQL via sqlc.
type CleanupRepo struct {
	credentialLogQ credentiallogdb.Querier
	webhookLogQ    webhooklogdb.Querier
	idempotencyQ   idempotencydb.Querier
}

// NewCleanupRepo creates a CleanupRepo.
func NewCleanupRepo(clq credentiallogdb.Querier, wlq webhooklogdb.Querier, iq idempotencydb.Querier) *CleanupRepo {
	return &CleanupRepo{
		credentialLogQ: clq,
		webhookLogQ:    wlq,
		idempotencyQ:   iq,
	}
}

// DeleteCredentialLogsOlderThan removes credential logs created before cutoff.
func (r *CleanupRepo) DeleteCredentialLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.credentialLogQ.DeleteCredentialLogsOlderThan(ctx, sql.NullTime{Time: cutoff, Valid: true})
	if err != nil {
		slog.Error("failed to delete old credential logs", "cutoff", cutoff, "error", err)
		return 0, domain.ErrInternal
	}
	return res.RowsAffected()
}

// DeleteWebhookLogsOlderThan removes webhook logs created before cutoff.
func (r *CleanupRepo) DeleteWebhookLogsOlderThan(ctx context.Context, cutoff time.Time) (int64, error) {
	res, err := r.webhookLogQ.DeleteWebhookLogsOlderThan(ctx, sql.NullTime{Time: cutoff, Valid: true})
	if err != nil {
		slog.Error("failed to delete old webhook logs", "cutoff", cutoff, "error", err)
		return 0, domain.ErrInternal
	}
	return res.RowsAffected()
}

var _ domain.CleanupRepository = (*CleanupRepo)(nil)

// DeleteExpiredIdempotencyKeys removes idempotency keys that have expired.
func (r *CleanupRepo) DeleteExpiredIdempotencyKeys(ctx context.Context) (int64, error) {
	res, err := r.idempotencyQ.DeleteExpiredIdempotencyKeys(ctx)
	if err != nil {
		slog.Error("failed to delete expired idempotency keys", "error", err)
		return 0, domain.ErrInternal
	}
	return res.RowsAffected()
}
