package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// ProductRepository defines persistence operations for the product domain.
type ProductRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input ProductInput) (*Product, error)
	GetByID(ctx context.Context, organizationID, productID uuid.UUID) (*Product, error)
	List(ctx context.Context, organizationID uuid.UUID) ([]Product, error)
	Update(ctx context.Context, organizationID, productID uuid.UUID, input ProductInput) (*Product, error)
	SoftDelete(ctx context.Context, organizationID, productID uuid.UUID) error
	HardDelete(ctx context.Context, organizationID, productID uuid.UUID) error
}

// Product represents a product entity in the domain layer.
type Product struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Active         bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// ProductInput represents the input for creating or updating a product.
type ProductInput struct {
	Name   string
	Active bool
}
