package service

import (
	"context"

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
	users      domain.UserRepository
	bcryptCost int
}

// NewUserService creates a UserService with database access.
// Uses bcrypt cost 12 by default (production-safe).
func NewUserService(users domain.UserRepository) *UserService {
	return &UserService{users: users, bcryptCost: 12}
}

// newUserServiceWithCost creates a UserService with a custom bcrypt cost (for testing).
func newUserServiceWithCost(users domain.UserRepository, cost int) *UserService {
	return &UserService{users: users, bcryptCost: cost}
}

// Create creates a new user within an organization.
// Returns ErrConflict if the email is already taken in the organization.
// Returns ErrForbidden if trying to create a second owner.
func (s *UserService) Create(ctx context.Context, organizationID string, req CreateUserRequest) (*domain.User, error) {
	tid, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, domain.ErrNotFound
	}

	// Check email uniqueness within organization
	_, err = s.users.GetByEmailAndOrg(ctx, tid, req.Email)
	if err == nil {
		return nil, domain.ErrConflict
	}

	// Block creation of a second owner
	if req.Role == domain.RoleOwner {
		existing, listErr := s.users.List(ctx, tid)
		if listErr != nil {
			return nil, listErr
		}
		for i := range existing {
			if existing[i].Role == domain.RoleOwner {
				return nil, domain.ErrForbidden
			}
		}
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), s.bcryptCost)
	if err != nil {
		return nil, err
	}

	return s.users.Create(ctx, tid, domain.CreateUserInput{
		Name:         req.Name,
		Email:        req.Email,
		PasswordHash: string(hash),
		Role:         string(req.Role),
	})
}

// List returns all active users for an organization, without password fields.
func (s *UserService) List(ctx context.Context, organizationID string) ([]domain.User, error) {
	tid, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return s.users.List(ctx, tid)
}

// GetByID returns a single user in an organization.
// Returns ErrNotFound if organizationID or userID are invalid or user does not exist.
func (s *UserService) GetByID(ctx context.Context, organizationID, userID string) (*domain.User, error) {
	tid, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return s.users.GetByID(ctx, tid, uid)
}

// Update updates a user's name and role within an organization.
// Returns ErrNotFound if the user does not exist in the organization.
func (s *UserService) Update(ctx context.Context, organizationID, userID string, req UpdateUserRequest) (*domain.User, error) {
	tid, err := uuid.Parse(organizationID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return nil, domain.ErrNotFound
	}
	return s.users.Update(ctx, tid, uid, domain.UpdateUserInput{
		Name: req.Name,
		Role: string(req.Role),
	})
}

// Delete soft-deletes a user within an organization.
// Returns ErrNotFound if the user does not exist in the organization.
func (s *UserService) Delete(ctx context.Context, organizationID, userID string) error {
	tid, err := uuid.Parse(organizationID)
	if err != nil {
		return domain.ErrNotFound
	}
	uid, err := uuid.Parse(userID)
	if err != nil {
		return domain.ErrNotFound
	}
	return s.users.SoftDelete(ctx, tid, uid)
}
