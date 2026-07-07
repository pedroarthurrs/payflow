package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// AddressRepository defines persistence operations for the address domain.
type AddressRepository interface {
	Create(ctx context.Context, organizationID, customerID uuid.UUID, input AddressInput) (*Address, error)
	GetByID(ctx context.Context, organizationID, customerID, addressID uuid.UUID) (*Address, error)
	List(ctx context.Context, organizationID, customerID uuid.UUID, params ListParams) ([]Address, error)
	Update(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input AddressInput) (*Address, error)
	Delete(ctx context.Context, organizationID, customerID, addressID uuid.UUID) error
}

// Address represents an address entity in the domain layer.
type Address struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	CustomerID     uuid.UUID
	Type           string
	PostalCode     string
	Street         string
	Number         *string
	Complement     *string
	Neighborhood   *string
	City           string
	State          string
	Country        string
	IsMain         bool
	CreatedAt      time.Time
}

// AddressInput represents the input for creating or updating an address.
type AddressInput struct {
	Type         *string
	PostalCode   string
	Street       string
	Number       *string
	Complement   *string
	Neighborhood *string
	City         string
	State        string
	Country      *string
	IsMain       *bool
}
