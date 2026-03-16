package domain

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// Customer represents a customer entity in the domain layer.
type Customer struct {
	ID           uuid.UUID
	TenantID     uuid.UUID
	Name         string
	Email        string
	Phone        *string
	Document     string
	DocumentType string
	Status       string
	Metadata     json.RawMessage
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// CreateCustomerInput represents the input for creating a new customer.
type CreateCustomerInput struct {
	Name         string
	Email        string
	Phone        *string
	Document     string
	DocumentType string
	Metadata     json.RawMessage
	Address      *CreateAddressInput
}

// UpdateCustomerInput represents the input for updating an existing customer.
type UpdateCustomerInput struct {
	Name         string
	Email        string
	Phone        *string
	Document     string
	DocumentType string
	Metadata     json.RawMessage
	Address      *CreateAddressInput
}
