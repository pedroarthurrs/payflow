package domain

import (
	"context"

	"github.com/google/uuid"
)

// AuthRepository defines persistence operations needed for authentication.
type AuthRepository interface {
	GetUserByEmail(ctx context.Context, email string) (*AuthUser, error)
	GetAPIKeyByHash(ctx context.Context, hash string) (*APIKey, error)
	UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error
}

// AuthUser holds the user data needed for authentication (includes password).
type AuthUser struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Password       string
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
//   - User mode: Email + Password (organization resolved from user record)
//   - System mode: APIKey
type LoginRequest struct {
	Email    string
	Password string
	APIKey   string
}
