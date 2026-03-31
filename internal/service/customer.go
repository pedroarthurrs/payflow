package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/sqlc-dev/pqtype"
)

// CustomerService handles customer business logic.
type CustomerService struct {
	db         *database.DB
	addressSvc *AddressService
}

// NewCustomerService creates a CustomerService.
func NewCustomerService(db *database.DB, addressSvc *AddressService) *CustomerService {
	return &CustomerService{db: db, addressSvc: addressSvc}
}

// Create creates a new customer for the given organization.
// When input.Address is provided, the customer and address are created atomically within a transaction.
func (s *CustomerService) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateCustomerInput) (*domain.Customer, *domain.Address, error) {
	if input.Address == nil {
		row, dbErr := s.db.Queries.CreateCustomer(ctx, dbgen.CreateCustomerParams{
			OrganizationID: organizationID,
			Name:           input.Name,
			Email:          input.Email,
			Phone:          toNullString(input.Phone),
			Document:       input.Document,
			DocumentType:   input.DocumentType,
			Metadata:       toNullRawMessage(input.Metadata),
		})
		if dbErr != nil {
			return nil, nil, s.handleWriteError(dbErr)
		}
		c := toDomainCustomer(row)
		return &c, nil, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return nil, nil, apierr.Internal("internal server error")
	}
	defer func() { _ = tx.Rollback() }()

	qtx := s.db.Queries.(*dbgen.Queries).WithTx(tx)

	row, dbErr := qtx.CreateCustomer(ctx, dbgen.CreateCustomerParams{
		OrganizationID: organizationID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          toNullString(input.Phone),
		Document:       input.Document,
		DocumentType:   input.DocumentType,
		Metadata:       toNullRawMessage(input.Metadata),
	})
	if dbErr != nil {
		return nil, nil, s.handleWriteError(dbErr)
	}

	c := toDomainCustomer(row)

	addr, addrErr := s.addressSvc.CreateWithTx(ctx, tx, organizationID, c.ID, *input.Address)
	if addrErr != nil {
		return nil, nil, addrErr
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return nil, nil, apierr.Internal("internal server error")
	}

	return &c, addr, nil
}

// List returns all active customers for the given organization.
func (s *CustomerService) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Customer, error) {
	rows, dbErr := s.db.Queries.ListCustomersByOrganization(ctx, organizationID)
	if dbErr != nil {
		slog.Error("failed to list customers", "organization_id", organizationID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	customers := make([]domain.Customer, len(rows))
	for i := range rows {
		customers[i] = toDomainCustomer(rows[i])
	}
	return customers, nil
}

// GetByID returns a single customer by ID for the given organization.
func (s *CustomerService) GetByID(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error) {
	row, dbErr := s.db.Queries.GetCustomerByID(ctx, dbgen.GetCustomerByIDParams{
		ID:             customerID,
		OrganizationID: organizationID,
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			return nil, domain.ErrCustomerNotFound
		}
		slog.Error("failed to get customer", "organization_id", organizationID, "customer_id", customerID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	c := toDomainCustomer(row)
	return &c, nil
}

// Update updates an existing customer for the given organization.
// When input.Address is provided, the customer update and address creation are performed atomically.
func (s *CustomerService) Update(ctx context.Context, organizationID, customerID uuid.UUID, input domain.UpdateCustomerInput) (*domain.Customer, *domain.Address, error) {
	if input.Address == nil {
		row, dbErr := s.db.Queries.UpdateCustomer(ctx, dbgen.UpdateCustomerParams{
			ID:             customerID,
			OrganizationID: organizationID,
			Name:           input.Name,
			Email:          input.Email,
			Phone:          toNullString(input.Phone),
			Document:       input.Document,
			DocumentType:   input.DocumentType,
			Metadata:       toNullRawMessage(input.Metadata),
		})
		if dbErr != nil {
			if errors.Is(dbErr, sql.ErrNoRows) {
				return nil, nil, domain.ErrCustomerNotFound
			}
			return nil, nil, s.handleWriteError(dbErr)
		}
		c := toDomainCustomer(row)
		return &c, nil, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin transaction", "error", err)
		return nil, nil, apierr.Internal("internal server error")
	}
	defer func() { _ = tx.Rollback() }()

	qtx := s.db.Queries.(*dbgen.Queries).WithTx(tx)

	row, dbErr := qtx.UpdateCustomer(ctx, dbgen.UpdateCustomerParams{
		ID:             customerID,
		OrganizationID: organizationID,
		Name:           input.Name,
		Email:          input.Email,
		Phone:          toNullString(input.Phone),
		Document:       input.Document,
		DocumentType:   input.DocumentType,
		Metadata:       toNullRawMessage(input.Metadata),
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			return nil, nil, domain.ErrCustomerNotFound
		}
		return nil, nil, s.handleWriteError(dbErr)
	}

	c := toDomainCustomer(row)

	addr, addrErr := s.addressSvc.CreateWithTx(ctx, tx, organizationID, customerID, *input.Address)
	if addrErr != nil {
		return nil, nil, addrErr
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit transaction", "error", err)
		return nil, nil, apierr.Internal("internal server error")
	}

	return &c, addr, nil
}

// Delete performs a soft delete on a customer for the given organization.
func (s *CustomerService) Delete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	// First verify the customer exists (SoftDeleteCustomer is :exec and won't return sql.ErrNoRows)
	_, getErr := s.db.Queries.GetCustomerByID(ctx, dbgen.GetCustomerByIDParams{
		ID:             customerID,
		OrganizationID: organizationID,
	})
	if getErr != nil {
		if errors.Is(getErr, sql.ErrNoRows) {
			return domain.ErrCustomerNotFound
		}
		slog.Error("failed to get customer for delete", "organization_id", organizationID, "customer_id", customerID, "error", getErr)
		return apierr.Internal("internal server error")
	}

	if dbErr := s.db.Queries.SoftDeleteCustomer(ctx, dbgen.SoftDeleteCustomerParams{
		ID:             customerID,
		OrganizationID: organizationID,
	}); dbErr != nil {
		slog.Error("failed to soft delete customer", "organization_id", organizationID, "customer_id", customerID, "error", dbErr)
		return apierr.Internal("internal server error")
	}

	return nil
}

// handleWriteError maps database write errors (unique constraint violations, etc.) to domain errors.
func (s *CustomerService) handleWriteError(dbErr error) error {
	var pqErr *pq.Error
	if errors.As(dbErr, &pqErr) && pqErr.Code == "23505" {
		switch pqErr.Constraint {
		case "unique_customer_email_per_organization":
			return apierr.Conflict("customer with this email already exists")
		case "unique_customer_doc_per_organization":
			return apierr.Conflict("customer with this document already exists")
		}
	}
	slog.Error("failed to write customer", "error", dbErr)
	return apierr.Internal("internal server error")
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

// toNullString converts a *string to sql.NullString.
func toNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// toNullRawMessage converts a json.RawMessage to pqtype.NullRawMessage.
func toNullRawMessage(data json.RawMessage) pqtype.NullRawMessage {
	if data == nil {
		return pqtype.NullRawMessage{}
	}
	return pqtype.NullRawMessage{RawMessage: data, Valid: true}
}

// HardDelete performs a physical deletion of a customer from the database.
// This is intended ONLY for specific situations such as:
//   - Legal requests (LGPD Art. 18, GDPR Art. 17 - Right to Erasure)
//   - Court orders requiring permanent data removal
//   - Compliance with data retention policies
//
// WARNING: This operation is IRREVERSIBLE and will PERMANENTLY delete the customer record.
// Use Delete() for normal operations (soft delete with deleted_at).
//
// Example usage for LGPD compliance:
//
//	// After receiving a legal request for data erasure
//	err := customerSvc.HardDelete(ctx, organizationID, customerID)
//	if err != nil {
//	    // Log the failure and notify compliance team
//	}
func (s *CustomerService) HardDelete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	if dbErr := s.db.Queries.HardDeleteCustomer(ctx, dbgen.HardDeleteCustomerParams{
		ID:             customerID,
		OrganizationID: organizationID,
	}); dbErr != nil {
		slog.Error("failed to hard delete customer", "organization_id", organizationID, "customer_id", customerID, "error", dbErr)
		return apierr.Internal("internal server error")
	}

	slog.Warn("customer permanently deleted", "organization_id", organizationID, "customer_id", customerID, "reason", "hard_delete")
	return nil
}
