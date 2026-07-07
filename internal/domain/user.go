package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Role represents a user's role within an organization.
type Role string

// Roles available within an organization.
// Hierarchy: owner (4) > admin (3) > developer (2) > support (1)
const (
	RoleOwner     Role = "owner"
	RoleAdmin     Role = "admin"
	RoleDeveloper Role = "developer"
	RoleSupport   Role = "support"
)

// RoleLevel returns the numeric level of the role for hierarchical comparison.
func RoleLevel(r Role) int {
	switch r {
	case RoleOwner:
		return 4
	case RoleAdmin:
		return 3
	case RoleDeveloper:
		return 2
	case RoleSupport:
		return 1
	default:
		return 0
	}
}

// UserAccount represents the global identity of a user, decoupled from any organization.
type UserAccount struct {
	ID        uuid.UUID
	Email     string
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
}

// OrganizationMember represents the link between a UserAccount and an Organization.
type OrganizationMember struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	UserAccountID  uuid.UUID
	Email          string // denormalized for display
	Name           string // denormalized for display
	Role           Role
	JoinedAt       time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// MemberPermission represents an individual permission grant or revocation for an organization member.
type MemberPermission struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID // desnormalizado de organization_members
	MemberID       uuid.UUID
	Permission     Permission
	Granted        bool // true=grant, false=revoke
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// OrganizationRef is a lightweight organization reference used for multi-org login listing.
type OrganizationRef struct {
	ID   uuid.UUID
	Name string
}

// UserAccountRepository defines persistence operations for global user accounts.
type UserAccountRepository interface {
	Create(ctx context.Context, input CreateUserAccountInput) (*UserAccount, error)
	GetByID(ctx context.Context, id uuid.UUID) (*UserAccount, error)
	GetByEmail(ctx context.Context, email string) (*UserAccount, error)
	Update(ctx context.Context, id uuid.UUID, input UpdateUserAccountInput) (*UserAccount, error)
	SoftDelete(ctx context.Context, id uuid.UUID) error
}

// MemberRepository defines persistence operations for organization membership.
type MemberRepository interface {
	AddMember(ctx context.Context, orgID, userAccountID uuid.UUID, role Role, invitedBy *uuid.UUID) (*OrganizationMember, error)
	GetMember(ctx context.Context, orgID, userAccountID uuid.UUID) (*OrganizationMember, error)
	ListMembers(ctx context.Context, orgID uuid.UUID, params ListParams) ([]OrganizationMember, int32, error)
	UpdateMemberRole(ctx context.Context, orgID, userAccountID uuid.UUID, role Role) (*OrganizationMember, error)
	RemoveMember(ctx context.Context, orgID, userAccountID uuid.UUID) error
	ListOrganizationsForUser(ctx context.Context, userAccountID uuid.UUID) ([]OrganizationRef, error)
}

// PermissionRepository — per-member permission grant/revoke overrides per organization.
type PermissionRepository interface {
	Upsert(ctx context.Context, memberID uuid.UUID, permission Permission, granted bool) error
	Delete(ctx context.Context, memberID uuid.UUID, permission Permission) error
	ListForMember(ctx context.Context, memberID uuid.UUID) ([]MemberPermission, error)
}

// CreateUserAccountInput is the input for creating a global user account.
type CreateUserAccountInput struct {
	Email        string
	Name         string
	PasswordHash string
}

// UpdateUserAccountInput is the input for updating a global user account.
type UpdateUserAccountInput struct {
	Name string
}

// MemberLogRecord represents an immutable log entry for a member action or auth event.
type MemberLogRecord struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	MemberID       uuid.UUID
	Action         string
	Method         *string
	Path           *string
	StatusCode     *int32
	LatencyMs      *int32
	IPAddress      *string
	UserAgent      *string
	Metadata       json.RawMessage
	CreatedAt      time.Time
}

// CreateMemberLogInput carries all fields needed to persist a new member log entry.
type CreateMemberLogInput struct {
	OrganizationID uuid.UUID
	MemberID       uuid.UUID
	Action         string
	Method         *string
	Path           *string
	StatusCode     *int32
	LatencyMs      *int32
	IPAddress      *string
	UserAgent      *string
	Metadata       json.RawMessage
}

// ListMemberLogsInput carries all filter and pagination parameters for listing member logs.
type ListMemberLogsInput struct {
	Limit      int
	Cursor     *CursorToken
	MemberID   *uuid.UUID
	Action     *string
	From       *time.Time
	To         *time.Time
	StatusCode *int32
	Method     *string
	PathPrefix *string
}

// MemberLogWriter defines the write-only persistence operation for member logs.
// Used by the MemberLog middleware.
type MemberLogWriter interface {
	Create(ctx context.Context, input CreateMemberLogInput) error
}

// MemberLogRepository defines read-only persistence operations for member logs.
type MemberLogRepository interface {
	List(ctx context.Context, orgID uuid.UUID, input ListMemberLogsInput) ([]MemberLogRecord, error)
	GetByID(ctx context.Context, orgID, logID uuid.UUID) (*MemberLogRecord, error)
}
