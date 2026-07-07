package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	productdb "payflow/internal/database/dbgen/product"
	"payflow/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// ProductRepo implements domain.ProductRepository backed by PostgreSQL via sqlc.
type ProductRepo struct {
	q productdb.Querier
}

// NewProductRepo creates a ProductRepo.
func NewProductRepo(q productdb.Querier) *ProductRepo {
	return &ProductRepo{q: q}
}

// Create persists a new product in an organization.
func (r *ProductRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.ProductInput) (*domain.Product, error) {
	row, err := r.q.CreateProduct(ctx, productdb.CreateProductParams{
		OrganizationID: organizationID,
		Name:           input.Name,
		Active:         sql.NullBool{Bool: input.Active, Valid: true},
	})
	if err != nil {
		return nil, mapProductWriteError(err)
	}
	c := toDomainProduct(row)
	return &c, nil
}

// List returns all products for an organization.
func (r *ProductRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Product, error) {
	rows, err := r.q.ListProductsByOrganization(ctx, productdb.ListProductsByOrganizationParams{
		OrganizationID: organizationID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list products", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	products := make([]domain.Product, len(rows))
	for i := range rows {
		products[i] = toDomainProduct(rows[i])
	}
	return products, nil
}

// GetByID retrieves a product by ID scoped to an organization.
func (r *ProductRepo) GetByID(ctx context.Context, organizationID, productID uuid.UUID) (*domain.Product, error) {
	row, err := r.q.GetProductByID(ctx, productdb.GetProductByIDParams{
		ID:             productID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		slog.Error("failed to get product", "organization_id", organizationID, "product_id", productID, "error", err)
		return nil, domain.ErrInternal
	}
	c := toDomainProduct(row)
	return &c, nil
}

// Update updates a product scoped to an organization.
func (r *ProductRepo) Update(ctx context.Context, organizationID, productID uuid.UUID, input domain.ProductInput) (*domain.Product, error) {
	row, err := r.q.UpdateProduct(ctx, productdb.UpdateProductParams{
		ID:             productID,
		OrganizationID: organizationID,
		Name:           input.Name,
		Active:         sql.NullBool{Bool: input.Active, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrProductNotFound
		}
		return nil, mapProductWriteError(err)
	}
	c := toDomainProduct(row)
	return &c, nil
}

// SoftDelete marks a product as deleted without removing the record.
func (r *ProductRepo) SoftDelete(ctx context.Context, organizationID, productID uuid.UUID) error {
	// Verify existence first (SoftDeleteProduct is :exec)
	_, err := r.q.GetProductByID(ctx, productdb.GetProductByIDParams{
		ID:             productID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrProductNotFound
		}
		slog.Error("failed to get product for delete", "organization_id", organizationID, "product_id", productID, "error", err)
		return domain.ErrInternal
	}

	if err := r.q.SoftDeleteProduct(ctx, productdb.SoftDeleteProductParams{
		ID:             productID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to soft delete product", "organization_id", organizationID, "product_id", productID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// HardDelete permanently removes a product from the organization.
func (r *ProductRepo) HardDelete(ctx context.Context, organizationID, productID uuid.UUID) error {
	if err := r.q.HardDeleteProduct(ctx, productdb.HardDeleteProductParams{
		ID:             productID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to hard delete product", "organization_id", organizationID, "product_id", productID, "error", err)
		return domain.ErrInternal
	}
	slog.Warn("product permanently deleted", "organization_id", organizationID, "product_id", productID, "reason", "hard_delete")
	return nil
}

// mapProductWriteError maps database write errors to domain errors.
func mapProductWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		return domain.ErrConflict
	}
	slog.Error("failed to write product", "error", err)
	return domain.ErrInternal
}

// toDomainProduct converts a productdb.Product to a domain.Product.
func toDomainProduct(row productdb.Product) domain.Product {
	c := domain.Product{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
	}
	if row.Active.Valid {
		c.Active = row.Active.Bool
	}
	if row.CreatedAt.Valid {
		c.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		c.UpdatedAt = row.UpdatedAt.Time
	}
	return c
}

var _ domain.ProductRepository = (*ProductRepo)(nil)
