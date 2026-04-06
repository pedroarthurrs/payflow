package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	credentialdb "i9pay/internal/database/dbgen/credential"
	"i9pay/internal/domain"

	"github.com/google/uuid"
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
func (r *CredentialRepo) List(ctx context.Context, orgID uuid.UUID) ([]domain.CredentialRecord, error) {
	rows, err := r.q.ListCredentials(ctx, orgID)
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
