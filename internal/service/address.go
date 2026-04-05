package service

import (
	"context"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// AddressService handles address business logic.
type AddressService struct {
	customers domain.CustomerRepository
	addresses domain.AddressRepository
}

// NewAddressService creates an AddressService.
func NewAddressService(customers domain.CustomerRepository, addresses domain.AddressRepository) *AddressService {
	return &AddressService{customers: customers, addresses: addresses}
}

// Create creates a new address for the given customer, applying defaults for omitted fields.
func (s *AddressService) Create(ctx context.Context, organizationID, customerID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, organizationID, customerID); err != nil {
		return nil, err
	}
	applyDefaults(&input)
	return s.addresses.Create(ctx, organizationID, customerID, input)
}

// List returns all addresses for the given customer.
func (s *AddressService) List(ctx context.Context, organizationID, customerID uuid.UUID) ([]domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, organizationID, customerID); err != nil {
		return nil, err
	}
	return s.addresses.List(ctx, organizationID, customerID)
}

// GetByID returns a single address by ID.
func (s *AddressService) GetByID(ctx context.Context, organizationID, customerID, addressID uuid.UUID) (*domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, organizationID, customerID); err != nil {
		return nil, err
	}
	return s.addresses.GetByID(ctx, organizationID, customerID, addressID)
}

// Update updates an existing address.
func (s *AddressService) Update(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
	if err := s.ensureCustomerExists(ctx, organizationID, customerID); err != nil {
		return nil, err
	}
	return s.addresses.Update(ctx, organizationID, customerID, addressID, input)
}

// Delete removes an address by ID.
func (s *AddressService) Delete(ctx context.Context, organizationID, customerID, addressID uuid.UUID) error {
	if err := s.ensureCustomerExists(ctx, organizationID, customerID); err != nil {
		return err
	}
	return s.addresses.Delete(ctx, organizationID, customerID, addressID)
}

// ensureCustomerExists checks that the customer exists for the given organization.
func (s *AddressService) ensureCustomerExists(ctx context.Context, organizationID, customerID uuid.UUID) error {
	_, err := s.customers.GetByID(ctx, organizationID, customerID)
	return err
}

// applyDefaults sets default values for omitted fields in CreateAddressInput.
func applyDefaults(input *domain.AddressInput) {
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
