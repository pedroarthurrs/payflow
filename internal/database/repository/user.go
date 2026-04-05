package repository

import (
	"context"
	"database/sql"
	"errors"

	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// UserRepo implements domain.UserRepository backed by PostgreSQL via sqlc.
type UserRepo struct {
	q dbgen.Querier
}

// NewUserRepo creates a UserRepo.
func NewUserRepo(q dbgen.Querier) *UserRepo {
	return &UserRepo{q: q}
}

// Create persists a new user in an organization.
func (r *UserRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateUserInput) (*domain.User, error) {
	row, err := r.q.CreateUser(ctx, dbgen.CreateUserParams{
		OrganizationID: organizationID,
		Name:           input.Name,
		Email:          input.Email,
		Password:       input.PasswordHash,
		Role:           input.Role,
	})
	if err != nil {
		return nil, err
	}
	u := mapUser(row)
	return &u, nil
}

// GetByID retrieves a user by ID scoped to an organization.
func (r *UserRepo) GetByID(ctx context.Context, organizationID, userID uuid.UUID) (*domain.User, error) {
	row, err := r.q.GetUserByID(ctx, dbgen.GetUserByIDParams{
		ID:             userID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u := mapUser(row)
	return &u, nil
}

// GetByEmailAndOrg retrieves a user by email within an organization.
func (r *UserRepo) GetByEmailAndOrg(ctx context.Context, organizationID uuid.UUID, email string) (*domain.User, error) {
	row, err := r.q.GetUserByEmailAndOrganization(ctx, dbgen.GetUserByEmailAndOrganizationParams{
		Email:          email,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u := mapUser(row)
	return &u, nil
}

// List returns all users for an organization.
func (r *UserRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.User, error) {
	rows, err := r.q.ListUsersByOrganization(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	users := make([]domain.User, len(rows))
	for i := range rows {
		users[i] = mapUser(rows[i])
	}
	return users, nil
}

// Update updates mutable user fields in an organization.
func (r *UserRepo) Update(ctx context.Context, organizationID, userID uuid.UUID, input domain.UpdateUserInput) (*domain.User, error) {
	row, err := r.q.UpdateUser(ctx, dbgen.UpdateUserParams{
		Name:           input.Name,
		Role:           input.Role,
		ID:             userID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}
	u := mapUser(row)
	return &u, nil
}

// SoftDelete marks a user as deleted in an organization.
func (r *UserRepo) SoftDelete(ctx context.Context, organizationID, userID uuid.UUID) error {
	// Verify existence first
	_, err := r.q.GetUserByID(ctx, dbgen.GetUserByIDParams{
		ID:             userID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return err
	}
	return r.q.SoftDeleteUser(ctx, dbgen.SoftDeleteUserParams{
		ID:             userID,
		OrganizationID: organizationID,
	})
}

// mapUser converts a dbgen.User to a domain.User (no password field).
func mapUser(u dbgen.User) domain.User {
	return domain.User{
		ID:             u.ID.String(),
		OrganizationID: u.OrganizationID.String(),
		Email:          u.Email,
		Name:           u.Name,
		Role:           domain.Role(u.Role),
	}
}

var _ domain.UserRepository = (*UserRepo)(nil)
