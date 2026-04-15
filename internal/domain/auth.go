package domain

import (
	"context"

	"github.com/google/uuid"
)

// AuthRepository defines persistence operations needed for authentication.
type AuthRepository interface {
	// GetUserAccountByEmail looks up a global user account by email.
	// Returns ErrUnauthorized-friendly errors when not found.
	GetUserAccountByEmail(ctx context.Context, email string) (*AuthUserAccount, error)
	// ListMembershipsForUser returns all org memberships for the given user account.
	ListMembershipsForUser(ctx context.Context, userAccountID uuid.UUID) ([]AuthMembership, error)
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error
}

// AuthUserAccount holds the global user account data needed for authentication.
type AuthUserAccount struct {
	ID       uuid.UUID
	Password string
}

// AuthMembership holds org-membership data returned during login.
type AuthMembership struct {
	MemberID       uuid.UUID
	OrganizationID uuid.UUID
	Name           string // Organization display name
	Role           Role
}

// APIKey holds API key data needed for authentication.
type APIKey struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
}

// AuthType represents the authentication mode used to obtain a token.
type AuthType string

// Authentication modes for token issuance.
const (
	AuthTypeUser   AuthType = "user"
	AuthTypeSystem AuthType = "system"
)

// LoginRequest represents the dynamic login payload.
// Exactly one mode must be populated:
//   - User mode: Email + Password (organization resolved from memberships)
//   - System mode: APIKey
type LoginRequest struct {
	Email    string
	Password string
	APIKey   string
}
