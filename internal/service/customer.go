package service

import (
	"context"
	"log/slog"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// CustomerService handles customer business logic.
type CustomerService struct {
	customers domain.CustomerRepository
	addresses domain.AddressRepository
	txManager domain.TxManager
}

// NewCustomerService creates a CustomerService.
func NewCustomerService(customers domain.CustomerRepository, addresses domain.AddressRepository, txManager domain.TxManager) *CustomerService {
	return &CustomerService{customers: customers, addresses: addresses, txManager: txManager}
}

// Create creates a new customer for the given organization.
// When input.Address is provided, the customer and address are created atomically within a transaction.
func (s *CustomerService) Create(ctx context.Context, organizationID uuid.UUID, input domain.CustomerInput) (*domain.Customer, *domain.Address, error) {
	if input.Address == nil {
		c, err := s.customers.Create(ctx, organizationID, input)
		if err != nil {
			return nil, nil, err
		}
		return c, nil, nil
	}

	var customer *domain.Customer
	var addr *domain.Address

	if err := s.txManager.WithTx(ctx, func(tx domain.TxHandle) error {
		c, err := tx.CustomerRepo().Create(ctx, organizationID, input)
		if err != nil {
			return err
		}
		customer = c

		applyDefaults(input.Address)
		a, err := tx.AddressRepo().Create(ctx, organizationID, c.ID, *input.Address)
		if err != nil {
			return err
		}
		addr = a
		return nil
	}); err != nil {
		return nil, nil, err
	}

	return customer, addr, nil
}

// List returns all active customers for the given organization.
func (s *CustomerService) List(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Customer, error) {
	return s.customers.List(ctx, organizationID, params)
}

// GetByID returns a single customer by ID for the given organization.
func (s *CustomerService) GetByID(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error) {
	return s.customers.GetByID(ctx, organizationID, customerID)
}

// Update updates an existing customer for the given organization.
// When input.Address is provided, the customer update and address creation are performed atomically.
func (s *CustomerService) Update(ctx context.Context, organizationID, customerID uuid.UUID, input domain.CustomerInput) (*domain.Customer, *domain.Address, error) {
	if input.Address == nil {
		c, err := s.customers.Update(ctx, organizationID, customerID, input)
		if err != nil {
			return nil, nil, err
		}
		return c, nil, nil
	}

	var customer *domain.Customer
	var addr *domain.Address

	if err := s.txManager.WithTx(ctx, func(tx domain.TxHandle) error {
		c, err := tx.CustomerRepo().Update(ctx, organizationID, customerID, input)
		if err != nil {
			return err
		}
		customer = c

		applyDefaults(input.Address)
		a, err := tx.AddressRepo().Create(ctx, organizationID, customerID, *input.Address)
		if err != nil {
			return err
		}
		addr = a
		return nil
	}); err != nil {
		return nil, nil, err
	}

	return customer, addr, nil
}

// Delete performs a soft delete on a customer for the given organization.
func (s *CustomerService) Delete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	return s.customers.SoftDelete(ctx, organizationID, customerID)
}

// HardDelete performs a physical deletion of a customer from the database.
// This is intended ONLY for specific situations such as:
//   - Legal requests (LGPD Art. 18, GDPR Art. 17 - Right to Erasure)
//   - Court orders requiring permanent data removal
//   - Compliance with data retention policies
//
// WARNING: This operation is IRREVERSIBLE and will PERMANENTLY delete the customer record.
// Use Delete() for normal operations (soft delete with deleted_at).
func (s *CustomerService) HardDelete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	if err := s.customers.HardDelete(ctx, organizationID, customerID); err != nil {
		return err
	}
	slog.Warn("customer permanently deleted", "organization_id", organizationID, "customer_id", customerID, "reason", "hard_delete")
	return nil
}
