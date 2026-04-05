package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// CustomerRepo implements domain.CustomerRepository backed by PostgreSQL via sqlc.
type CustomerRepo struct {
	q dbgen.Querier
}

// NewCustomerRepo creates a CustomerRepo.
func NewCustomerRepo(q dbgen.Querier) *CustomerRepo {
	return &CustomerRepo{q: q}
}

// Create persists a new customer in an organization.
func (r *CustomerRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
	row, err := r.q.CreateCustomer(ctx, dbgen.CreateCustomerParams{
		OrganizationID: organizationID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          toNullString(input.Phone),
		Document:       input.Document,
		DocumentType:   input.DocumentType,
		Metadata:       toNullRawMessage(input.Metadata),
	})
	if err != nil {
		return nil, mapCustomerWriteError(err)
	}
	c := toDomainCustomer(row)
	return &c, nil
}

// GetByID retrieves a customer by ID scoped to an organization.
func (r *CustomerRepo) GetByID(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error) {
	row, err := r.q.GetCustomerByID(ctx, dbgen.GetCustomerByIDParams{
		ID:             customerID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		slog.Error("failed to get customer", "organization_id", organizationID, "customer_id", customerID, "error", err)
		return nil, domain.ErrInternal
	}
	c := toDomainCustomer(row)
	return &c, nil
}

// List returns all customers for an organization.
func (r *CustomerRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Customer, error) {
	rows, err := r.q.ListCustomersByOrganization(ctx, organizationID)
	if err != nil {
		slog.Error("failed to list customers", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	customers := make([]domain.Customer, len(rows))
	for i := range rows {
		customers[i] = toDomainCustomer(rows[i])
	}
	return customers, nil
}

// Update updates a customer scoped to an organization.
func (r *CustomerRepo) Update(ctx context.Context, organizationID, customerID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
	row, err := r.q.UpdateCustomer(ctx, dbgen.UpdateCustomerParams{
		ID:             customerID,
		OrganizationID: organizationID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          toNullString(input.Phone),
		Document:       input.Document,
		DocumentType:   input.DocumentType,
		Metadata:       toNullRawMessage(input.Metadata),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		return nil, mapCustomerWriteError(err)
	}
	c := toDomainCustomer(row)
	return &c, nil
}

// SoftDelete marks a customer as deleted without removing the record.
func (r *CustomerRepo) SoftDelete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	// Verify existence first (SoftDeleteCustomer is :exec)
	_, err := r.q.GetCustomerByID(ctx, dbgen.GetCustomerByIDParams{
		ID:             customerID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrCustomerNotFound
		}
		slog.Error("failed to get customer for delete", "organization_id", organizationID, "customer_id", customerID, "error", err)
		return domain.ErrInternal
	}

	if err := r.q.SoftDeleteCustomer(ctx, dbgen.SoftDeleteCustomerParams{
		ID:             customerID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to soft delete customer", "organization_id", organizationID, "customer_id", customerID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// HardDelete permanently removes a customer from the organization.
func (r *CustomerRepo) HardDelete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	if err := r.q.HardDeleteCustomer(ctx, dbgen.HardDeleteCustomerParams{
		ID:             customerID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to hard delete customer", "organization_id", organizationID, "customer_id", customerID, "error", err)
		return domain.ErrInternal
	}
	slog.Warn("customer permanently deleted", "organization_id", organizationID, "customer_id", customerID, "reason", "hard_delete")
	return nil
}

// mapCustomerWriteError maps database write errors to domain errors.
func mapCustomerWriteError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		switch pqErr.Constraint {
		case "unique_customer_email_per_organization":
			return domain.ErrDuplicateEmail
		case "unique_customer_doc_per_organization":
			return domain.ErrDuplicateDocument
		}
	}
	slog.Error("failed to write customer", "error", err)
	return domain.ErrInternal
}

// toDomainCustomer converts a dbgen.Customer to a domain.Customer.
func toDomainCustomer(row dbgen.Customer) domain.Customer {
	c := domain.Customer{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Email:          row.Email,
		Document:       row.Document,
		DocumentType:   row.DocumentType,
	}
	if row.Phone.Valid {
		c.Phone = &row.Phone.String
	}
	if row.Status.Valid {
		c.Status = row.Status.String
	}
	if row.Metadata.Valid {
		c.Metadata = json.RawMessage(row.Metadata.RawMessage)
	}
	if row.CreatedAt.Valid {
		c.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		c.UpdatedAt = row.UpdatedAt.Time
	}
	return c
}

var _ domain.CustomerRepository = (*CustomerRepo)(nil)
