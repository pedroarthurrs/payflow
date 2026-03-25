package domain

// Role represents a user's role within a tenant.
type Role string

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

// AuthType represents the authentication mode used to obtain a token.
type AuthType string

const (
	AuthTypeUser   AuthType = "user"
	AuthTypeSystem AuthType = "system"
)

// User represents a human user with access to a tenant's panel.
type User struct {
	ID       string
	TenantID string
	Email    string
	Name     string
	Role     Role
}

// LoginRequest represents the dynamic login payload.
// Exactly one mode must be populated:
//   - User mode: Email + Password (tenant resolved from user record)
//   - System mode: APIKey
type LoginRequest struct {
	Email    string
	Password string
	APIKey   string
}
