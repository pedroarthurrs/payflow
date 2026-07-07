package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	idempotencydb "payflow/internal/database/dbgen/idempotency"
	"payflow/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// IdempotencyRepo implements domain.IdempotencyStorage backed by PostgreSQL via sqlc.
type IdempotencyRepo struct {
	q idempotencydb.Querier
}

// NewIdempotencyRepo creates an IdempotencyRepo.
func NewIdempotencyRepo(q idempotencydb.Querier) *IdempotencyRepo {
	return &IdempotencyRepo{q: q}
}

// Check looks up an existing idempotency record for the given organization and key.
// Returns nil when no valid (non-expired) record is found.
func (r *IdempotencyRepo) Check(ctx context.Context, key string, orgID uuid.UUID) (*domain.IdempotencyRecord, error) {
	row, err := r.q.GetIdempotencyKey(ctx, idempotencydb.GetIdempotencyKeyParams{
		OrganizationID: orgID,
		IdempotencyKey: key,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		slog.Error("failed to check idempotency key", "organization_id", orgID, "key", key, "error", err)
		return nil, domain.ErrInternal
	}

	if !row.ResponseCode.Valid {
		return nil, nil
	}

	var body []byte
	if row.ResponseBody.Valid {
		body = row.ResponseBody.RawMessage
	}

	return &domain.IdempotencyRecord{
		ResponseCode: int(row.ResponseCode.Int32),
		ResponseBody: body,
	}, nil
}

// Save persists (upserts) an idempotency record with the calculated expiration.
func (r *IdempotencyRepo) Save(ctx context.Context, key string, orgID uuid.UUID, requestPath string, record domain.IdempotencyRecord, ttl time.Duration) error {
	params := idempotencydb.SaveIdempotencyKeyParams{
		OrganizationID: orgID,
		IdempotencyKey: key,
		RequestPath:    requestPath,
		ResponseCode:   sql.NullInt32{Int32: int32(record.ResponseCode), Valid: true},
		ExpiresAt:      time.Now().Add(ttl),
	}

	if record.ResponseBody != nil {
		params.ResponseBody = pqtype.NullRawMessage{RawMessage: record.ResponseBody, Valid: true}
	}

	if err := r.q.SaveIdempotencyKey(ctx, params); err != nil {
		slog.Error("failed to save idempotency key", "organization_id", orgID, "key", key, "error", err)
		return domain.ErrInternal
	}
	return nil
}

var _ domain.IdempotencyStorage = (*IdempotencyRepo)(nil)
