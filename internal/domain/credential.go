package domain

import (
	"context"
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
	List(ctx context.Context, orgID uuid.UUID) ([]CredentialRecord, error)
	SoftDelete(ctx context.Context, orgID, credentialID uuid.UUID) error
}
