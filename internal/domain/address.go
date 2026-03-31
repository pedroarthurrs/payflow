package domain

import (
	"time"

	"github.com/google/uuid"
)

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

// CreateAddressInput represents the input for creating a new address.
type CreateAddressInput struct {
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

// UpdateAddressInput represents the input for updating an existing address.
type UpdateAddressInput struct {
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
