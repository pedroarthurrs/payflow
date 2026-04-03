package service

import (
	"context"
	"errors"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ============================================================
// Mock CustomerRepository
// ============================================================

type mockCustomerRepo struct {
	createFn     func(ctx context.Context, organizationID uuid.UUID, input domain.CreateCustomerInput) (*domain.Customer, error)
	getByIDFn    func(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error)
	listFn       func(ctx context.Context, organizationID uuid.UUID) ([]domain.Customer, error)
	updateFn     func(ctx context.Context, organizationID, customerID uuid.UUID, input domain.UpdateCustomerInput) (*domain.Customer, error)
	softDeleteFn func(ctx context.Context, organizationID, customerID uuid.UUID) error
	hardDeleteFn func(ctx context.Context, organizationID, customerID uuid.UUID) error
}

var _ domain.CustomerRepository = (*mockCustomerRepo)(nil)

func (m *mockCustomerRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateCustomerInput) (*domain.Customer, error) {
	if m.createFn != nil {
		return m.createFn(ctx, organizationID, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCustomerRepo) GetByID(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, organizationID, customerID)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCustomerRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.Customer, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID)
	}
	return nil, nil
}

func (m *mockCustomerRepo) Update(ctx context.Context, organizationID, customerID uuid.UUID, input domain.UpdateCustomerInput) (*domain.Customer, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, organizationID, customerID, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCustomerRepo) SoftDelete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, organizationID, customerID)
	}
	return nil
}

func (m *mockCustomerRepo) HardDelete(ctx context.Context, organizationID, customerID uuid.UUID) error {
	if m.hardDeleteFn != nil {
		return m.hardDeleteFn(ctx, organizationID, customerID)
	}
	return nil
}

// ============================================================
// Mock AddressRepository
// ============================================================

type mockAddressRepo struct {
	createFn  func(ctx context.Context, organizationID, customerID uuid.UUID, input domain.CreateAddressInput) (*domain.Address, error)
	getByIDFn func(ctx context.Context, organizationID, customerID, addressID uuid.UUID) (*domain.Address, error)
	listFn    func(ctx context.Context, organizationID, customerID uuid.UUID) ([]domain.Address, error)
	updateFn  func(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input domain.UpdateAddressInput) (*domain.Address, error)
	deleteFn  func(ctx context.Context, organizationID, customerID, addressID uuid.UUID) error
}

var _ domain.AddressRepository = (*mockAddressRepo)(nil)

func (m *mockAddressRepo) Create(ctx context.Context, organizationID, customerID uuid.UUID, input domain.CreateAddressInput) (*domain.Address, error) {
	if m.createFn != nil {
		return m.createFn(ctx, organizationID, customerID, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockAddressRepo) GetByID(ctx context.Context, organizationID, customerID, addressID uuid.UUID) (*domain.Address, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, organizationID, customerID, addressID)
	}
	return nil, domain.ErrAddressNotFound
}

func (m *mockAddressRepo) List(ctx context.Context, organizationID, customerID uuid.UUID) ([]domain.Address, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID, customerID)
	}
	return nil, nil
}

func (m *mockAddressRepo) Update(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input domain.UpdateAddressInput) (*domain.Address, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, organizationID, customerID, addressID, input)
	}
	return nil, domain.ErrAddressNotFound
}

func (m *mockAddressRepo) Delete(ctx context.Context, organizationID, customerID, addressID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, organizationID, customerID, addressID)
	}
	return nil
}

// ============================================================
// Mock UserRepository
// ============================================================

type mockUserRepo struct {
	createFn           func(ctx context.Context, organizationID uuid.UUID, name, email, passwordHash, role string) (*domain.User, error)
	getByIDFn          func(ctx context.Context, organizationID, userID uuid.UUID) (*domain.User, error)
	getByEmailAndOrgFn func(ctx context.Context, organizationID uuid.UUID, email string) (*domain.User, error)
	listFn             func(ctx context.Context, organizationID uuid.UUID) ([]domain.User, error)
	updateFn           func(ctx context.Context, organizationID, userID uuid.UUID, name, role string) (*domain.User, error)
	softDeleteFn       func(ctx context.Context, organizationID, userID uuid.UUID) error
}

var _ domain.UserRepository = (*mockUserRepo)(nil)

func (m *mockUserRepo) Create(ctx context.Context, organizationID uuid.UUID, name, email, passwordHash, role string) (*domain.User, error) {
	if m.createFn != nil {
		return m.createFn(ctx, organizationID, name, email, passwordHash, role)
	}
	return nil, errors.New("not implemented")
}

func (m *mockUserRepo) GetByID(ctx context.Context, organizationID, userID uuid.UUID) (*domain.User, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, organizationID, userID)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) GetByEmailAndOrg(ctx context.Context, organizationID uuid.UUID, email string) (*domain.User, error) {
	if m.getByEmailAndOrgFn != nil {
		return m.getByEmailAndOrgFn(ctx, organizationID, email)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.User, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID)
	}
	return nil, nil
}

func (m *mockUserRepo) Update(ctx context.Context, organizationID, userID uuid.UUID, name, role string) (*domain.User, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, organizationID, userID, name, role)
	}
	return nil, errors.New("not found")
}

func (m *mockUserRepo) SoftDelete(ctx context.Context, organizationID, userID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, organizationID, userID)
	}
	return nil
}

// ============================================================
// Mock AuthRepository
// ============================================================

type mockAuthRepo struct {
	getUserByEmailFn       func(ctx context.Context, email string) (*domain.AuthUser, error)
	getAPIKeyByHashFn      func(ctx context.Context, hash string) (*domain.APIKey, error)
	updateAPIKeyLastUsedFn func(ctx context.Context, id uuid.UUID) error
	lastUsedUpdated        bool
}

var _ domain.AuthRepository = (*mockAuthRepo)(nil)

func (m *mockAuthRepo) GetUserByEmail(ctx context.Context, email string) (*domain.AuthUser, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return nil, errors.New("not found")
}

func (m *mockAuthRepo) GetAPIKeyByHash(ctx context.Context, hash string) (*domain.APIKey, error) {
	if m.getAPIKeyByHashFn != nil {
		return m.getAPIKeyByHashFn(ctx, hash)
	}
	return nil, errors.New("not found")
}

func (m *mockAuthRepo) UpdateAPIKeyLastUsed(ctx context.Context, id uuid.UUID) error {
	m.lastUsedUpdated = true
	if m.updateAPIKeyLastUsedFn != nil {
		return m.updateAPIKeyLastUsedFn(ctx, id)
	}
	return nil
}

// ============================================================
// Mock TxManager
// ============================================================

type mockTxManager struct {
	withTxFn func(ctx context.Context, fn func(tx domain.TxHandle) error) error
}

var _ domain.TxManager = (*mockTxManager)(nil)

func (m *mockTxManager) WithTx(ctx context.Context, fn func(tx domain.TxHandle) error) error {
	if m.withTxFn != nil {
		return m.withTxFn(ctx, fn)
	}
	return fn(&mockTxHandle{})
}

// ============================================================
// Mock TxHandle
// ============================================================

type mockTxHandle struct {
	customers domain.CustomerRepository
	addresses domain.AddressRepository
}

var _ domain.TxHandle = (*mockTxHandle)(nil)

func (m *mockTxHandle) CustomerRepo() domain.CustomerRepository {
	if m.customers != nil {
		return m.customers
	}
	return &mockCustomerRepo{}
}

func (m *mockTxHandle) AddressRepo() domain.AddressRepository {
	if m.addresses != nil {
		return m.addresses
	}
	return &mockAddressRepo{}
}

// ============================================================
// Shared helpers
// ============================================================

// customerExistsMock returns a GetByID function that always finds the customer.
func customerExistsMock() func(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error) {
	return func(_ context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error) {
		return &domain.Customer{ID: customerID, OrganizationID: organizationID}, nil
	}
}
