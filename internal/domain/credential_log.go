package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

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

// CredentialLogRepository defines read-only persistence operations for credential logs.
type CredentialLogRepository interface {
	List(ctx context.Context, orgID uuid.UUID, input ListCredentialLogsInput) ([]CredentialLogRecord, error)
	GetByID(ctx context.Context, orgID, logID uuid.UUID) (*CredentialLogRecord, error)
}
