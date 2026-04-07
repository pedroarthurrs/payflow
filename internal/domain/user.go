package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines persistence operations for the user domain.
type UserRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateUserInput) (*User, error)
	GetByID(ctx context.Context, organizationID, userID uuid.UUID) (*User, error)
	GetByEmailAndOrg(ctx context.Context, organizationID uuid.UUID, email string) (*User, error)
	List(ctx context.Context, organizationID uuid.UUID, params ListParams) ([]User, error)
	Update(ctx context.Context, organizationID, userID uuid.UUID, input UpdateUserInput) (*User, error)
	SoftDelete(ctx context.Context, organizationID, userID uuid.UUID) error
}

// CreateUserInput represents the input for creating a new user.
type CreateUserInput struct {
	Name         string
	Email        string
	PasswordHash string
	Role         string
}

// UpdateUserInput represents the input for updating an existing user.
type UpdateUserInput struct {
	Name string
	Role string
}

// Role represents a user's role within an organization.
type Role string

// Roles available within an organization.
const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleMember Role = "member"
)

// RoleLevel retorna o nível numérico do papel para comparação hierárquica.
func RoleLevel(r Role) int {
	switch r {
	case RoleOwner:
		return 3
	case RoleAdmin:
		return 2
	case RoleMember:
		return 1
	default:
		return 0
	}
}

// User represents a human user with access to an organization's panel.
type User struct {
	ID             string
	OrganizationID string
	Email          string
	Name           string
	Role           Role
}
