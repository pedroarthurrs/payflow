package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CredentialRecord represents a Credential without the raw_key (never stored).
type CredentialRecord struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	KeyPrefix      string
	Environment    string // "test" | "production"
	LastUsedAt     *time.Time
	ExpiresAt      *time.Time
	CreatedAt      time.Time
}

// CreateCredentialInput is the input for persistence (without raw_key).
type CreateCredentialInput struct {
	Name        string
	KeyHash     string
	KeyPrefix   string
	Environment string
	ExpiresAt   *time.Time
}

// CredentialRepository defines persistence operations for the credential domain.
type CredentialRepository interface {
	Create(ctx context.Context, orgID uuid.UUID, input CreateCredentialInput) (*CredentialRecord, error)
	GetByID(ctx context.Context, orgID, credentialID uuid.UUID) (*CredentialRecord, error)
	List(ctx context.Context, orgID uuid.UUID, params ListParams) ([]CredentialRecord, error)
	SoftDelete(ctx context.Context, orgID, credentialID uuid.UUID) error
}

// CredentialLogRecord represents an immutable log entry for a credential-authenticated request.
type CredentialLogRecord struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CredentialID   *uuid.UUID
	Method         *string
	Path           *string
	RequestBody    json.RawMessage
	ResponseBody   json.RawMessage
	StatusCode     *int32
	LatencyMs      *int32
	IPAddress      *string
	UserAgent      *string
	CreatedAt      time.Time
}

// CursorToken holds the decoded keyset pagination cursor values.
type CursorToken struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

// ListParams holds common offset pagination parameters for list queries.
type ListParams struct {
	Limit  int32
	Offset int32
}

// ListCredentialLogsInput carries all filter and pagination parameters for listing logs.
type ListCredentialLogsInput struct {
	Limit      int
	Cursor     *CursorToken
	From       *time.Time
	To         *time.Time
	StatusCode *int32
	Method     *string
	PathPrefix *string
}

// CreateCredentialLogInput carries all fields needed to persist a new credential log entry.
type CreateCredentialLogInput struct {
	OrganizationID uuid.UUID
	CredentialID   *uuid.UUID
	Method         *string
	Path           *string
	RequestBody    json.RawMessage
	ResponseBody   json.RawMessage
	StatusCode     *int32
	LatencyMs      *int32
	IPAddress      *string
	UserAgent      *string
}

// CredentialLogWriter defines the write-only persistence operation for credential logs.
type CredentialLogWriter interface {
	Create(ctx context.Context, input CreateCredentialLogInput) error
}

// CredentialLogRepository defines read-only persistence operations for credential logs.
type CredentialLogRepository interface {
	List(ctx context.Context, orgID uuid.UUID, input ListCredentialLogsInput) ([]CredentialLogRecord, error)
	GetByID(ctx context.Context, orgID, logID uuid.UUID) (*CredentialLogRecord, error)
}
