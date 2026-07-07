package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	addressdb "payflow/internal/database/dbgen/address"
	"payflow/internal/domain"

	"github.com/google/uuid"
)

// AddressRepo implements domain.AddressRepository backed by PostgreSQL via sqlc.
type AddressRepo struct {
	q addressdb.Querier
}

// NewAddressRepo creates an AddressRepo.
func NewAddressRepo(q addressdb.Querier) *AddressRepo {
	return &AddressRepo{q: q}
}

// Create persists a new address for a customer in an organization.
func (r *AddressRepo) Create(ctx context.Context, organizationID, customerID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
	row, err := r.q.CreateAddress(ctx, addressdb.CreateAddressParams{
		OrganizationID: organizationID,
		CustomerID:     customerID,
		Type:           toNullAddressType(input.Type),
		PostalCode:     input.PostalCode,
		Street:         input.Street,
		Number:         toNullString(input.Number),
		Complement:     toNullString(input.Complement),
		Neighborhood:   toNullString(input.Neighborhood),
		City:           input.City,
		State:          input.State,
		Country:        toNullString(input.Country),
		IsMain:         toNullBool(input.IsMain),
	})
	if err != nil {
		slog.Error("failed to create address", "organization_id", organizationID, "customer_id", customerID, "error", err)
		return nil, domain.ErrInternal
	}
	addr := toDomainAddress(row)
	return &addr, nil
}

// GetByID retrieves an address by ID scoped to organization and customer.
func (r *AddressRepo) GetByID(ctx context.Context, organizationID, customerID, addressID uuid.UUID) (*domain.Address, error) {
	row, err := r.q.GetAddressByID(ctx, addressdb.GetAddressByIDParams{
		ID:             addressID,
		CustomerID:     customerID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		slog.Error("failed to get address", "organization_id", organizationID, "address_id", addressID, "error", err)
		return nil, domain.ErrInternal
	}
	addr := toDomainAddress(row)
	return &addr, nil
}

// List returns all addresses for a customer in an organization.
func (r *AddressRepo) List(ctx context.Context, organizationID, customerID uuid.UUID, params domain.ListParams) ([]domain.Address, error) {
	rows, err := r.q.ListAddressesByCustomer(ctx, addressdb.ListAddressesByCustomerParams{
		CustomerID:     customerID,
		OrganizationID: organizationID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list addresses", "organization_id", organizationID, "customer_id", customerID, "error", err)
		return nil, domain.ErrInternal
	}
	addresses := make([]domain.Address, len(rows))
	for i := range rows {
		addresses[i] = toDomainAddress(rows[i])
	}
	return addresses, nil
}

// Update updates an existing address scoped to organization and customer.
func (r *AddressRepo) Update(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
	row, err := r.q.UpdateAddress(ctx, addressdb.UpdateAddressParams{
		ID:             addressID,
		CustomerID:     customerID,
		OrganizationID: organizationID,
		Type:           toNullAddressType(input.Type),
		PostalCode:     input.PostalCode,
		Street:         input.Street,
		Number:         toNullString(input.Number),
		Complement:     toNullString(input.Complement),
		Neighborhood:   toNullString(input.Neighborhood),
		City:           input.City,
		State:          input.State,
		Country:        toNullString(input.Country),
		IsMain:         toNullBool(input.IsMain),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		slog.Error("failed to update address", "organization_id", organizationID, "address_id", addressID, "error", err)
		return nil, domain.ErrInternal
	}
	addr := toDomainAddress(row)
	return &addr, nil
}

// Delete removes an address scoped to organization and customer.
func (r *AddressRepo) Delete(ctx context.Context, organizationID, customerID, addressID uuid.UUID) error {
	// Verify existence first (DeleteAddress is :exec)
	_, err := r.q.GetAddressByID(ctx, addressdb.GetAddressByIDParams{
		ID:             addressID,
		CustomerID:     customerID,
		OrganizationID: organizationID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ErrAddressNotFound
		}
		slog.Error("failed to get address for delete", "organization_id", organizationID, "address_id", addressID, "error", err)
		return domain.ErrInternal
	}

	if err := r.q.DeleteAddress(ctx, addressdb.DeleteAddressParams{
		ID:             addressID,
		CustomerID:     customerID,
		OrganizationID: organizationID,
	}); err != nil {
		slog.Error("failed to delete address", "organization_id", organizationID, "address_id", addressID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// toDomainAddress converts a addressdb.Address to a domain.Address.
func toDomainAddress(row addressdb.Address) domain.Address {
	a := domain.Address{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		CustomerID:     row.CustomerID,
		PostalCode:     row.PostalCode,
		Street:         row.Street,
		City:           row.City,
		State:          row.State,
	}
	if row.Type.Valid {
		a.Type = string(row.Type.AddressType)
	}
	if row.Number.Valid {
		a.Number = &row.Number.String
	}
	if row.Complement.Valid {
		a.Complement = &row.Complement.String
	}
	if row.Neighborhood.Valid {
		a.Neighborhood = &row.Neighborhood.String
	}
	if row.Country.Valid {
		a.Country = row.Country.String
	}
	if row.IsMain.Valid {
		a.IsMain = row.IsMain.Bool
	}
	if row.CreatedAt.Valid {
		a.CreatedAt = row.CreatedAt.Time
	}
	return a
}

var _ domain.AddressRepository = (*AddressRepo)(nil)
