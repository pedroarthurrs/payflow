package service

import (
	"context"
	"log/slog"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ProductService handles product business logic.
type ProductService struct {
	products domain.ProductRepository
}

// NewProductService creates a ProductService.
func NewProductService(products domain.ProductRepository) *ProductService {
	return &ProductService{products: products}
}

// Create creates a new product for the given organization.
func (s *ProductService) Create(ctx context.Context, organizationID uuid.UUID, input domain.ProductInput) (*domain.Product, error) {
	return s.products.Create(ctx, organizationID, input)
}

// List returns all active products for the given organization.
func (s *ProductService) List(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Product, error) {
	return s.products.List(ctx, organizationID, params)
}

// GetByID returns a single product by ID for the given organization.
func (s *ProductService) GetByID(ctx context.Context, organizationID, productID uuid.UUID) (*domain.Product, error) {
	return s.products.GetByID(ctx, organizationID, productID)
}

// Update updates an existing product for the given organization.
func (s *ProductService) Update(ctx context.Context, organizationID, productID uuid.UUID, input domain.ProductInput) (*domain.Product, error) {
	return s.products.Update(ctx, organizationID, productID, input)
}

// Delete performs a soft delete on a product for the given organization.
func (s *ProductService) Delete(ctx context.Context, organizationID, productID uuid.UUID) error {
	return s.products.SoftDelete(ctx, organizationID, productID)
}

// HardDelete performs a physical deletion of a product from the database.
func (s *ProductService) HardDelete(ctx context.Context, organizationID, productID uuid.UUID) error {
	if err := s.products.HardDelete(ctx, organizationID, productID); err != nil {
		return err
	}
	slog.Warn("product permanently deleted", "organization_id", organizationID, "product_id", productID, "reason", "hard_delete")
	return nil
}
