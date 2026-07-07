package domain

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// CustomerRepository defines persistence operations for the customer domain.
type CustomerRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CustomerInput) (*Customer, error)
	GetByID(ctx context.Context, organizationID, customerID uuid.UUID) (*Customer, error)
	List(ctx context.Context, organizationID uuid.UUID, params ListParams) ([]Customer, error)
	Update(ctx context.Context, organizationID, customerID uuid.UUID, input CustomerInput) (*Customer, error)
	SoftDelete(ctx context.Context, organizationID, customerID uuid.UUID) error
	HardDelete(ctx context.Context, organizationID, customerID uuid.UUID) error
}

// Customer represents a customer entity in the domain layer.
type Customer struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Email          string
	Phone          *string
	Document       string
	DocumentType   string
	Status         string
	Metadata       json.RawMessage
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CustomerInput represents the input for creating or updating a customer.
type CustomerInput struct {
	Name         string
	Email        string
	Phone        *string
	Document     string
	DocumentType string
	Metadata     json.RawMessage
	Address      *AddressInput
}
