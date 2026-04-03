package domain

import (
	"context"

	"github.com/google/uuid"
)

// UserRepository defines persistence operations for the user domain.
type UserRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, name, email, passwordHash, role string) (*User, error)
	GetByID(ctx context.Context, organizationID, userID uuid.UUID) (*User, error)
	GetByEmailAndOrg(ctx context.Context, organizationID uuid.UUID, email string) (*User, error)
	List(ctx context.Context, organizationID uuid.UUID) ([]User, error)
	Update(ctx context.Context, organizationID, userID uuid.UUID, name, role string) (*User, error)
	SoftDelete(ctx context.Context, organizationID, userID uuid.UUID) error
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
