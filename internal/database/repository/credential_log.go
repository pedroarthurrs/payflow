package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	credentiallogdb "i9pay/internal/database/dbgen/credentiallog"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// CredentialLogRepo implements domain.CredentialLogRepository backed by PostgreSQL via sqlc.
type CredentialLogRepo struct {
	q credentiallogdb.Querier
}

// NewCredentialLogRepo creates a CredentialLogRepo.
func NewCredentialLogRepo(q credentiallogdb.Querier) *CredentialLogRepo {
	return &CredentialLogRepo{q: q}
}

// List returns a page of credential logs for an organization, applying optional filters.
func (r *CredentialLogRepo) List(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
	params := credentiallogdb.ListCredentialLogsParams{
		OrganizationID: orgID,
		PageLimit:      int32(input.Limit),
	}

	if input.Cursor != nil {
		params.CursorTs = sql.NullTime{Time: input.Cursor.CreatedAt, Valid: true}
		params.CursorID = uuid.NullUUID{UUID: input.Cursor.ID, Valid: true}
	}
	if input.From != nil {
		params.FromTs = sql.NullTime{Time: *input.From, Valid: true}
	}
	if input.To != nil {
		params.ToTs = sql.NullTime{Time: *input.To, Valid: true}
	}
	if input.StatusCode != nil {
		params.StatusCode = sql.NullInt32{Int32: *input.StatusCode, Valid: true}
	}
	if input.Method != nil {
		params.Method = sql.NullString{String: *input.Method, Valid: true}
	}
	if input.PathPrefix != nil {
		params.PathPrefix = sql.NullString{String: *input.PathPrefix, Valid: true}
	}

	rows, err := r.q.ListCredentialLogs(ctx, params)
	if err != nil {
		slog.Error("failed to list credential logs", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}

	records := make([]domain.CredentialLogRecord, len(rows))
	for i := range rows {
		records[i] = toDomainCredentialLog(rows[i])
	}
	return records, nil
}

// GetByID retrieves a single credential log by ID scoped to an organization.
func (r *CredentialLogRepo) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error) {
	row, err := r.q.GetCredentialLogByID(ctx, credentiallogdb.GetCredentialLogByIDParams{
		OrganizationID: orgID,
		ID:             logID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCredentialLogNotFound
		}
		slog.Error("failed to get credential log", "organization_id", orgID, "log_id", logID, "error", err)
		return nil, domain.ErrInternal
	}
	rec := toDomainCredentialLog(row)
	return &rec, nil
}

func toDomainCredentialLog(row credentiallogdb.CredentialLog) domain.CredentialLogRecord {
	rec := domain.CredentialLogRecord{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
	}
	if row.CredentialID.Valid {
		id := row.CredentialID.UUID
		rec.CredentialID = &id
	}
	if row.Method.Valid {
		rec.Method = &row.Method.String
	}
	if row.Path.Valid {
		rec.Path = &row.Path.String
	}
	if row.RequestBody.Valid {
		rec.RequestBody = row.RequestBody.RawMessage
	}
	if row.ResponseBody.Valid {
		rec.ResponseBody = row.ResponseBody.RawMessage
	}
	if row.StatusCode.Valid {
		rec.StatusCode = &row.StatusCode.Int32
	}
	if row.LatencyMs.Valid {
		rec.LatencyMs = &row.LatencyMs.Int32
	}
	if row.IpAddress.Valid {
		rec.IPAddress = &row.IpAddress.String
	}
	if row.UserAgent.Valid {
		rec.UserAgent = &row.UserAgent.String
	}
	if row.CreatedAt.Valid {
		rec.CreatedAt = row.CreatedAt.Time
	}
	return rec
}

var _ domain.CredentialLogRepository = (*CredentialLogRepo)(nil)
