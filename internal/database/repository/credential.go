package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	credentialdb "i9pay/internal/database/dbgen/credential"
	credentiallogdb "i9pay/internal/database/dbgen/credentiallog"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// CredentialRepo implements domain.CredentialRepository backed by PostgreSQL via sqlc.
type CredentialRepo struct {
	q credentialdb.Querier
}

// NewCredentialRepo creates a CredentialRepo.
func NewCredentialRepo(q credentialdb.Querier) *CredentialRepo {
	return &CredentialRepo{q: q}
}

// Create persists a new credential for an organization.
func (r *CredentialRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
	row, err := r.q.CreateCredential(ctx, credentialdb.CreateCredentialParams{
		OrganizationID: orgID,
		Name:           input.Name,
		KeyHash:        input.KeyHash,
		KeyPrefix:      input.KeyPrefix,
		Environment:    credentialdb.EnvironmentType(input.Environment),
		ExpiresAt:      toNullTime(input.ExpiresAt),
	})
	if err != nil {
		slog.Error("failed to create credential", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}
	rec := toDomainCredential(row)
	return &rec, nil
}

// GetByID retrieves a credential by ID scoped to an organization.
func (r *CredentialRepo) GetByID(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error) {
	row, err := r.q.GetCredentialByID(ctx, credentialdb.GetCredentialByIDParams{
		OrganizationID: orgID,
		ID:             credentialID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCredentialNotFound
		}
		slog.Error("failed to get credential", "organization_id", orgID, "credential_id", credentialID, "error", err)
		return nil, domain.ErrInternal
	}
	rec := toDomainCredentialFromGet(row)
	return &rec, nil
}

// List returns all active credentials for an organization.
func (r *CredentialRepo) List(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.CredentialRecord, error) {
	rows, err := r.q.ListCredentials(ctx, credentialdb.ListCredentialsParams{
		OrganizationID: orgID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list credentials", "organization_id", orgID, "error", err)
		return nil, domain.ErrInternal
	}
	records := make([]domain.CredentialRecord, len(rows))
	for i := range rows {
		records[i] = toDomainCredentialFromList(rows[i])
	}
	return records, nil
}

// SoftDelete marks a credential as deleted without removing the record.
func (r *CredentialRepo) SoftDelete(ctx context.Context, orgID, credentialID uuid.UUID) error {
	_, err := r.q.GetCredentialByID(ctx, credentialdb.GetCredentialByIDParams{
		OrganizationID: orgID,
		ID:             credentialID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrCredentialNotFound
		}
		slog.Error("failed to get credential for delete", "organization_id", orgID, "credential_id", credentialID, "error", err)
		return domain.ErrInternal
	}

	if err := r.q.SoftDeleteCredential(ctx, credentialdb.SoftDeleteCredentialParams{
		OrganizationID: orgID,
		ID:             credentialID,
	}); err != nil {
		slog.Error("failed to soft delete credential", "organization_id", orgID, "credential_id", credentialID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

func toDomainCredential(row credentialdb.CreateCredentialRow) domain.CredentialRecord {
	rec := domain.CredentialRecord{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		KeyPrefix:      row.KeyPrefix,
		Environment:    string(row.Environment),
	}
	if row.LastUsedAt.Valid {
		t := row.LastUsedAt.Time
		rec.LastUsedAt = &t
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		rec.ExpiresAt = &t
	}
	if row.CreatedAt.Valid {
		rec.CreatedAt = row.CreatedAt.Time
	}
	return rec
}

func toDomainCredentialFromList(row credentialdb.ListCredentialsRow) domain.CredentialRecord {
	rec := domain.CredentialRecord{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		KeyPrefix:      row.KeyPrefix,
		Environment:    string(row.Environment),
	}
	if row.LastUsedAt.Valid {
		t := row.LastUsedAt.Time
		rec.LastUsedAt = &t
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		rec.ExpiresAt = &t
	}
	if row.CreatedAt.Valid {
		rec.CreatedAt = row.CreatedAt.Time
	}
	return rec
}

func toDomainCredentialFromGet(row credentialdb.GetCredentialByIDRow) domain.CredentialRecord {
	rec := domain.CredentialRecord{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		KeyPrefix:      row.KeyPrefix,
		Environment:    string(row.Environment),
	}
	if row.LastUsedAt.Valid {
		t := row.LastUsedAt.Time
		rec.LastUsedAt = &t
	}
	if row.ExpiresAt.Valid {
		t := row.ExpiresAt.Time
		rec.ExpiresAt = &t
	}
	if row.CreatedAt.Valid {
		rec.CreatedAt = row.CreatedAt.Time
	}
	return rec
}

// toNullTime converts a *time.Time to sql.NullTime.
func toNullTime(t *time.Time) sql.NullTime {
	if t == nil {
		return sql.NullTime{}
	}
	return sql.NullTime{Time: *t, Valid: true}
}

var _ domain.CredentialRepository = (*CredentialRepo)(nil)

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
		records[i] = toDomainCredentialLogFromListRow(rows[i])
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
	rec := toDomainCredentialLogFromGetRow(row)
	return &rec, nil
}

func toDomainCredentialLogFromGetRow(row credentiallogdb.GetCredentialLogByIDRow) domain.CredentialLogRecord {
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

func toDomainCredentialLogFromListRow(row credentiallogdb.ListCredentialLogsRow) domain.CredentialLogRecord {
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

// Create persists a new credential log entry.
func (r *CredentialLogRepo) Create(ctx context.Context, input domain.CreateCredentialLogInput) error {
	params := credentiallogdb.CreateCredentialLogParams{
		OrganizationID: input.OrganizationID,
	}
	if input.CredentialID != nil {
		params.CredentialID = uuid.NullUUID{UUID: *input.CredentialID, Valid: true}
	}
	if input.Method != nil {
		params.Method = sql.NullString{String: *input.Method, Valid: true}
	}
	if input.Path != nil {
		params.Path = sql.NullString{String: *input.Path, Valid: true}
	}
	if input.RequestBody != nil {
		params.RequestBody = pqtype.NullRawMessage{RawMessage: input.RequestBody, Valid: true}
	}
	if input.ResponseBody != nil {
		params.ResponseBody = pqtype.NullRawMessage{RawMessage: input.ResponseBody, Valid: true}
	}
	if input.StatusCode != nil {
		params.StatusCode = sql.NullInt32{Int32: *input.StatusCode, Valid: true}
	}
	if input.LatencyMs != nil {
		params.LatencyMs = sql.NullInt32{Int32: *input.LatencyMs, Valid: true}
	}
	if input.IPAddress != nil {
		params.IpAddress = sql.NullString{String: *input.IPAddress, Valid: true}
	}
	if input.UserAgent != nil {
		params.UserAgent = sql.NullString{String: *input.UserAgent, Valid: true}
	}

	if err := r.q.CreateCredentialLog(ctx, params); err != nil {
		slog.Error("failed to create credential log", "organization_id", input.OrganizationID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

var _ domain.CredentialLogWriter = (*CredentialLogRepo)(nil)
