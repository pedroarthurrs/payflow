package repository

import (
	"context"
	"database/sql"
	"errors"

	authdb "i9pay/internal/database/dbgen/auth"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// AuthRepo implements domain.AuthRepository backed by PostgreSQL via sqlc.
type AuthRepo struct {
	q authdb.Querier
}

// NewAuthRepo creates an AuthRepo.
func NewAuthRepo(q authdb.Querier) *AuthRepo {
	return &AuthRepo{q: q}
}

// GetUserByEmail returns auth user data by email.
func (r *AuthRepo) GetUserByEmail(ctx context.Context, email string) (*domain.AuthUser, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrUnauthorized
		}
		return nil, err
	}
	return &domain.AuthUser{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Password:       row.Password,
		Role:           domain.Role(row.Role),
	}, nil
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
