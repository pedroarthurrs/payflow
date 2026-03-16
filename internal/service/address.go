package service

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/database"
	"i9pay/internal/database/dbgen"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// AddressService handles address business logic.
type AddressService struct {
	db *database.DB
}

// NewAddressService creates an AddressService.
func NewAddressService(db *database.DB) *AddressService {
	return &AddressService{db: db}
}

// Create creates a new address for the given customer, applying defaults for omitted fields.
func (s *AddressService) Create(ctx context.Context, tenantID, customerID uuid.UUID, input domain.CreateAddressInput) (*domain.Address, error) {
	// Verify customer exists
	if err := s.ensureCustomerExists(ctx, tenantID, customerID); err != nil {
		return nil, err
	}

	applyDefaults(&input)

	row, dbErr := s.db.Queries.CreateAddress(ctx, dbgen.CreateAddressParams{
		TenantID:     tenantID,
		CustomerID:   customerID,
		Type:         toNullAddressType(input.Type),
		PostalCode:   input.PostalCode,
		Street:       input.Street,
		Number:       toNullString(input.Number),
		Complement:   toNullString(input.Complement),
		Neighborhood: toNullString(input.Neighborhood),
		City:         input.City,
		State:        input.State,
		Country:      toNullString(input.Country),
		IsMain:       toNullBool(input.IsMain),
	})
	if dbErr != nil {
		slog.Error("failed to create address", "tenant_id", tenantID, "customer_id", customerID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	addr := toDomainAddress(row)
	return &addr, nil
}

// CreateWithTx creates an address within the provided transaction, applying defaults.
func (s *AddressService) CreateWithTx(ctx context.Context, tx *sql.Tx, tenantID, customerID uuid.UUID, input domain.CreateAddressInput) (*domain.Address, error) {
	applyDefaults(&input)

	qtx := s.db.Queries.(*dbgen.Queries).WithTx(tx)

	row, dbErr := qtx.CreateAddress(ctx, dbgen.CreateAddressParams{
		TenantID:     tenantID,
		CustomerID:   customerID,
		Type:         toNullAddressType(input.Type),
		PostalCode:   input.PostalCode,
		Street:       input.Street,
		Number:       toNullString(input.Number),
		Complement:   toNullString(input.Complement),
		Neighborhood: toNullString(input.Neighborhood),
		City:         input.City,
		State:        input.State,
		Country:      toNullString(input.Country),
		IsMain:       toNullBool(input.IsMain),
	})
	if dbErr != nil {
		slog.Error("failed to create address in tx", "tenant_id", tenantID, "customer_id", customerID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	addr := toDomainAddress(row)
	return &addr, nil
}

// List returns all addresses for the given customer.
func (s *AddressService) List(ctx context.Context, tenantID, customerID uuid.UUID) ([]domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, tenantID, customerID); err != nil {
		return nil, err
	}

	rows, dbErr := s.db.Queries.ListAddressesByCustomer(ctx, dbgen.ListAddressesByCustomerParams{
		CustomerID: customerID,
		TenantID:   tenantID,
	})
	if dbErr != nil {
		slog.Error("failed to list addresses", "tenant_id", tenantID, "customer_id", customerID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	addresses := make([]domain.Address, len(rows))
	for i, row := range rows {
		addresses[i] = toDomainAddress(row)
	}
	return addresses, nil
}

// GetByID returns a single address by ID.
func (s *AddressService) GetByID(ctx context.Context, tenantID, customerID, addressID uuid.UUID) (*domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, tenantID, customerID); err != nil {
		return nil, err
	}

	row, dbErr := s.db.Queries.GetAddressByID(ctx, dbgen.GetAddressByIDParams{
		ID:         addressID,
		CustomerID: customerID,
		TenantID:   tenantID,
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		slog.Error("failed to get address", "tenant_id", tenantID, "address_id", addressID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	addr := toDomainAddress(row)
	return &addr, nil
}

// Update updates an existing address.
func (s *AddressService) Update(ctx context.Context, tenantID, customerID, addressID uuid.UUID, input domain.UpdateAddressInput) (*domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, tenantID, customerID); err != nil {
		return nil, err
	}

	row, dbErr := s.db.Queries.UpdateAddress(ctx, dbgen.UpdateAddressParams{
		ID:           addressID,
		CustomerID:   customerID,
		TenantID:     tenantID,
		Type:         toNullAddressType(input.Type),
		PostalCode:   input.PostalCode,
		Street:       input.Street,
		Number:       toNullString(input.Number),
		Complement:   toNullString(input.Complement),
		Neighborhood: toNullString(input.Neighborhood),
		City:         input.City,
		State:        input.State,
		Country:      toNullString(input.Country),
		IsMain:       toNullBool(input.IsMain),
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			return nil, domain.ErrAddressNotFound
		}
		slog.Error("failed to update address", "tenant_id", tenantID, "address_id", addressID, "error", dbErr)
		return nil, apierr.Internal("internal server error")
	}

	addr := toDomainAddress(row)
	return &addr, nil
}

// Delete removes an address by ID.
func (s *AddressService) Delete(ctx context.Context, tenantID, customerID, addressID uuid.UUID) error {
	if err := s.ensureCustomerExists(ctx, tenantID, customerID); err != nil {
		return err
	}

	// Verify address exists first (DeleteAddress is :exec and won't return sql.ErrNoRows)
	_, getErr := s.db.Queries.GetAddressByID(ctx, dbgen.GetAddressByIDParams{
		ID:         addressID,
		CustomerID: customerID,
		TenantID:   tenantID,
	})
	if getErr != nil {
		if errors.Is(getErr, sql.ErrNoRows) {
			return domain.ErrAddressNotFound
		}
		slog.Error("failed to get address for delete", "tenant_id", tenantID, "address_id", addressID, "error", getErr)
		return apierr.Internal("internal server error")
	}

	if dbErr := s.db.Queries.DeleteAddress(ctx, dbgen.DeleteAddressParams{
		ID:         addressID,
		CustomerID: customerID,
		TenantID:   tenantID,
	}); dbErr != nil {
		slog.Error("failed to delete address", "tenant_id", tenantID, "address_id", addressID, "error", dbErr)
		return apierr.Internal("internal server error")
	}

	return nil
}

// ensureCustomerExists checks that the customer exists for the given tenant.
func (s *AddressService) ensureCustomerExists(ctx context.Context, tenantID, customerID uuid.UUID) error {
	_, dbErr := s.db.Queries.GetCustomerByID(ctx, dbgen.GetCustomerByIDParams{
		ID:       customerID,
		TenantID: tenantID,
	})
	if dbErr != nil {
		if errors.Is(dbErr, sql.ErrNoRows) {
			return domain.ErrCustomerNotFound
		}
		slog.Error("failed to verify customer", "tenant_id", tenantID, "customer_id", customerID, "error", dbErr)
		return apierr.Internal("internal server error")
	}
	return nil
}

// toDomainAddress converts a dbgen.Address to a domain.Address.
func toDomainAddress(row dbgen.Address) domain.Address {
	a := domain.Address{
		ID:         row.ID,
		TenantID:   row.TenantID,
		CustomerID: row.CustomerID,
		PostalCode: row.PostalCode,
		Street:     row.Street,
		City:       row.City,
		State:      row.State,
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

// applyDefaults sets default values for omitted fields in CreateAddressInput.
func applyDefaults(input *domain.CreateAddressInput) {
	if input.Type == nil {
		billing := "billing"
		input.Type = &billing
	}
	if input.Country == nil {
		br := "BR"
		input.Country = &br
	}
	if input.IsMain == nil {
		t := true
		input.IsMain = &t
	}
}

// toNullAddressType converts a *string to dbgen.NullAddressType.
func toNullAddressType(s *string) dbgen.NullAddressType {
	if s == nil {
		return dbgen.NullAddressType{}
	}
	return dbgen.NullAddressType{AddressType: dbgen.AddressType(*s), Valid: true}
}

// toNullBool converts a *bool to sql.NullBool.
func toNullBool(b *bool) sql.NullBool {
	if b == nil {
		return sql.NullBool{}
	}
	return sql.NullBool{Bool: *b, Valid: true}
}
