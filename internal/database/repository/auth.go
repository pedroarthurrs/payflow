package repository

import (
	"context"
	"database/sql"
	"errors"

	authdb "i9pay/internal/database/dbgen/auth"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// AuthRepo implements domain.AuthRepository backed by PostgreSQL via sqlc + raw SQL.
type AuthRepo struct {
	q  authdb.Querier
	db *sql.DB
}

// NewAuthRepo creates an AuthRepo.
func NewAuthRepo(q authdb.Querier, db *sql.DB) *AuthRepo {
	return &AuthRepo{q: q, db: db}
}

// GetUserAccountByEmail returns the global user account (user table) by email.
func (r *AuthRepo) GetUserAccountByEmail(ctx context.Context, email string) (*domain.AuthUserAccount, error) {
	const query = `
		SELECT id, password FROM "user"
		WHERE email = $1 AND deleted_at IS NULL
		LIMIT 1`
	row := r.db.QueryRowContext(ctx, query, email)
	var acc domain.AuthUserAccount
	if err := row.Scan(&acc.ID, &acc.Password); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	return &acc, nil
}

// ListMembershipsForUser returns all active org memberships for a user account
// including the organization's display name.
func (r *AuthRepo) ListMembershipsForUser(ctx context.Context, userAccountID uuid.UUID) ([]domain.AuthMembership, error) {
	const query = `
		SELECT om.id, om.organization_id, o.name, om.role
		FROM organization_members om
		JOIN organizations o ON o.id = om.organization_id
		WHERE om.user_id = $1
		  AND om.deleted_at IS NULL
		  AND o.deleted_at IS NULL`

	rows, err := r.db.QueryContext(ctx, query, userAccountID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var memberships []domain.AuthMembership
	for rows.Next() {
		var m domain.AuthMembership
		if err := rows.Scan(&m.MemberID, &m.OrganizationID, &m.Name, &m.Role); err != nil {
			return nil, err
		}
		memberships = append(memberships, m)
	}
	return memberships, rows.Err()
}

// GetAPIKeyByHash returns an API key by its hashed value.
func (r *AuthRepo) GetAPIKeyByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	row, err := r.q.GetAPIKeyByHash(ctx, hash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	return &domain.APIKey{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
	}, nil
}

// UpdateAPIKeyLastUsed updates the last-used timestamp for an API key.
func (r *AuthRepo) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	return r.q.UpdateAPIKeyLastUsed(ctx, id)
}

var _ domain.AuthRepository = (*AuthRepo)(nil)
