package service

import (
	"context"
	"database/sql"
	"errors"

	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

// CreateUserRequest holds the data needed to create a new user.
type CreateUserRequest struct {
	Name     string
	Email    string
	Password string
	Role     domain.Role
}

// UpdateUserRequest holds the data needed to update an existing user.
type UpdateUserRequest struct {
	Name string
	Role domain.Role
}

// UserService handles user management logic.
type UserService struct {
	db           *database.DB
	bcryptCost   int
}

// NewUserService creates a UserService with database access.
// Uses bcrypt cost 12 by default (production-safe).
func NewUserService(db *database.DB) *UserService {
	return &UserService{db: db, bcryptCost: 12}
}

// newUserServiceWithCost creates a UserService with a custom bcrypt cost (for testing).
func newUserServiceWithCost(db *database.DB, cost int) *UserService {
	return &UserService{db: db, bcryptCost: cost}
}

// Create creates a new user within a tenant.
// Returns ErrConflict if the email is already taken in the tenant.
// Returns ErrForbidden if trying to create a second owner.
func (s *UserService) Create(ctx context.Context, tenantID string, req CreateUserRequest) (*domain.User, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// Check email uniqueness within tenant
	_, err = s.db.Queries.GetUserByEmailAndTenant(ctx, dbgen.GetUserByEmailAndTenantParams{
		Email:    req.Email,
		TenantID: tid,
	})
	if err == nil {
		// User found — email already taken
		return nil, domain.ErrConflict
	}

	// Block creation of a second owner
	if req.Role == domain.RoleOwner {
		existing, listErr := s.db.Queries.ListUsersByTenant(ctx, tid)
		if listErr != nil {
			return nil, listErr
		}
		for _, u := range existing {
			if u.Role == string(domain.RoleOwner) {
				return nil, domain.ErrForbidden
			}
		}
	}

	// Hash password with bcrypt cost 12
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, err
	}

	row, err := s.db.Queries.CreateUser(ctx, dbgen.CreateUserParams{
		TenantID: tid,
		Name:     req.Name,
		Email:    req.Email,
		Password: string(hash),
		Role:     string(req.Role),
	})
	if err != nil {
		return nil, err
	}

	return mapUser(row), nil
}

// List returns all active users for a tenant, without password fields.
func (s *UserService) List(ctx context.Context, tenantID string) ([]domain.User, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	rows, err := s.db.Queries.ListUsersByTenant(ctx, tid)
	if err != nil {
		return nil, err
	}

	users := make([]domain.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, *mapUser(row))
	}
	return users, nil
}

// Update updates a user's name and role within a tenant.
// Returns ErrNotFound if the user does not exist in the tenant.
func (s *UserService) Update(ctx context.Context, tenantID, userID string, req UpdateUserRequest) (*domain.User, error) {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	row, err := s.db.Queries.UpdateUser(ctx, dbgen.UpdateUserParams{
		Name:     req.Name,
		Role:     string(req.Role),
		ID:       uid,
		TenantID: tid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrNotFound
		}
		return nil, err
	}

	return mapUser(row), nil
}

// Delete soft-deletes a user within a tenant.
// Returns ErrNotFound if the user does not exist in the tenant.
func (s *UserService) Delete(ctx context.Context, tenantID, userID string) error {
	tid, err := uuid.Parse(tenantID)
	if err != nil {
		return domain.ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.ErrNotFound
	}

	// Check existence before soft-deleting
	_, err = s.db.Queries.GetUserByID(ctx, dbgen.GetUserByIDParams{
		ID:       uid,
		TenantID: tid,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrNotFound
		}
		return domain.ErrNotFound
	}

	return s.db.Queries.SoftDeleteUser(ctx, dbgen.SoftDeleteUserParams{
		ID:       uid,
		TenantID: tid,
	})
}

// mapUser converts a dbgen.User to a domain.User (no password field).
func mapUser(u dbgen.User) *domain.User {
	return &domain.User{
		ID:       u.ID.String(),
		TenantID: u.TenantID.String(),
		Email:    u.Email,
		Name:     u.Name,
		Role:     domain.Role(u.Role),
	}
}
