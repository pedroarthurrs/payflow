package service

import (
	"context"
	"errors"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ============================================================
// Mock CustomerRepository
// ============================================================

type mockCustomerRepo struct {
	createFn     func(ctx context.Context, organizationID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error)
	getByIDFn    func(ctx context.Context, organizationID, customerID uuid.UUID) (*domain.Customer, error)
	listFn       func(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Customer, error)
	updateFn     func(ctx context.Context, organizationID, customerID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error)
	softDeleteFn func(ctx context.Context, organizationID, customerID uuid.UUID) error
	hardDeleteFn func(ctx context.Context, organizationID, customerID uuid.UUID) error
}

var _ domain.CustomerRepository = (*mockCustomerRepo)(nil)

func (m *mockCustomerRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
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

func (m *mockCustomerRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Customer, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID, params)
	}
	return nil, nil
}

func (m *mockCustomerRepo) Update(ctx context.Context, organizationID, customerID uuid.UUID, input domain.CustomerInput) (*domain.Customer, error) {
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
	createFn  func(ctx context.Context, organizationID, customerID uuid.UUID, input domain.AddressInput) (*domain.Address, error)
	getByIDFn func(ctx context.Context, organizationID, customerID, addressID uuid.UUID) (*domain.Address, error)
	listFn    func(ctx context.Context, organizationID, customerID uuid.UUID, params domain.ListParams) ([]domain.Address, error)
	updateFn  func(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input domain.AddressInput) (*domain.Address, error)
	deleteFn  func(ctx context.Context, organizationID, customerID, addressID uuid.UUID) error
}

var _ domain.AddressRepository = (*mockAddressRepo)(nil)

func (m *mockAddressRepo) Create(ctx context.Context, organizationID, customerID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
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

func (m *mockAddressRepo) List(ctx context.Context, organizationID, customerID uuid.UUID, params domain.ListParams) ([]domain.Address, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID, customerID, params)
	}
	return nil, nil
}

func (m *mockAddressRepo) Update(ctx context.Context, organizationID, customerID, addressID uuid.UUID, input domain.AddressInput) (*domain.Address, error) {
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
// Mock AuthRepository
// ============================================================

type mockAuthRepo struct {
	getUserAccountByEmailFn  func(ctx context.Context, email string) (*domain.AuthUserAccount, error)
	listMembershipsForUserFn func(ctx context.Context, userAccountID uuid.UUID) ([]domain.AuthMembership, error)
	getAPIKeyByHashFn        func(ctx context.Context, hash string) (*domain.APIKey, error)
	updateAPIKeyLastUsedFn   func(ctx context.Context, id uuid.UUID) error
	lastUsedUpdated          bool
}

var _ domain.AuthRepository = (*mockAuthRepo)(nil)

func (m *mockAuthRepo) GetUserAccountByEmail(ctx context.Context, email string) (*domain.AuthUserAccount, error) {
	if m.getUserAccountByEmailFn != nil {
		return m.getUserAccountByEmailFn(ctx, email)
	}
	return nil, errors.New("not found")
}

func (m *mockAuthRepo) ListMembershipsForUser(ctx context.Context, userAccountID uuid.UUID) ([]domain.AuthMembership, error) {
	if m.listMembershipsForUserFn != nil {
		return m.listMembershipsForUserFn(ctx, userAccountID)
	}
	return nil, nil
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
	customers   domain.CustomerRepository
	addresses   domain.AddressRepository
	credentials domain.CredentialRepository
	paymentFSM  domain.PaymentFSMRepository
	subs        domain.SubscriptionRepository
	subItems    domain.SubscriptionItemRepository
	subSchedule domain.SubscriptionScheduleRepository
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

func (m *mockTxHandle) CredentialRepo() domain.CredentialRepository {
	if m.credentials != nil {
		return m.credentials
	}
	return &mockCredentialRepo{}
}

func (m *mockTxHandle) PaymentFSMRepo() domain.PaymentFSMRepository {
	if m.paymentFSM != nil {
		return m.paymentFSM
	}
	return &mockPaymentFSMRepo{}
}

func (m *mockTxHandle) SubscriptionRepo() domain.SubscriptionRepository {
	if m.subs != nil {
		return m.subs
	}
	return &mockSubscriptionRepoStub{}
}

func (m *mockTxHandle) SubscriptionItemRepo() domain.SubscriptionItemRepository {
	if m.subItems != nil {
		return m.subItems
	}
	return &mockSubscriptionItemRepoStub{}
}

func (m *mockTxHandle) SubscriptionScheduleRepo() domain.SubscriptionScheduleRepository {
	if m.subSchedule != nil {
		return m.subSchedule
	}
	return &mockSubscriptionScheduleRepoStub{}
}

// Stub implementations for TxHandle that delegate to the outer mock when needed.

type mockSubscriptionRepoStub struct{}

func (s *mockSubscriptionRepoStub) Create(_ context.Context, _ uuid.UUID, _ domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	return &domain.Subscription{ID: uuid.New()}, nil
}
func (s *mockSubscriptionRepoStub) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
	return nil, domain.ErrSubscriptionNotFound
}
func (s *mockSubscriptionRepoStub) List(_ context.Context, _ uuid.UUID, _ domain.SubscriptionListParams) ([]domain.Subscription, error) {
	return nil, nil
}
func (s *mockSubscriptionRepoStub) Count(_ context.Context, _ uuid.UUID, _ domain.SubscriptionListParams) (int64, error) {
	return 0, nil
}
func (s *mockSubscriptionRepoStub) UpdateStatus(_ context.Context, _, _ uuid.UUID, _ domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
	return nil, nil
}
func (s *mockSubscriptionRepoStub) SetCancelAtPeriodEnd(_ context.Context, _, _ uuid.UUID, _ bool) (*domain.Subscription, error) {
	return nil, nil
}
func (s *mockSubscriptionRepoStub) AdvancePeriod(_ context.Context, _, _ uuid.UUID, _, _ time.Time) (*domain.Subscription, error) {
	return nil, nil
}

type mockSubscriptionItemRepoStub struct{}

func (s *mockSubscriptionItemRepoStub) CreateItems(_ context.Context, _, _ uuid.UUID, _ []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	return nil, nil
}
func (s *mockSubscriptionItemRepoStub) ReplaceItems(_ context.Context, _, _ uuid.UUID, _ []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	return nil, nil
}
func (s *mockSubscriptionItemRepoStub) ListBySubscription(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
	return nil, nil
}

type mockSubscriptionScheduleRepoStub struct{}

func (s *mockSubscriptionScheduleRepoStub) Create(_ context.Context, _ uuid.UUID, _ domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
	return nil, nil
}
func (s *mockSubscriptionScheduleRepoStub) GetPending(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionSchedule, error) {
	return nil, domain.ErrScheduleNotFound
}
func (s *mockSubscriptionScheduleRepoStub) Cancel(_ context.Context, _, _ uuid.UUID) error {
	return nil
}

// ============================================================
// Mock PaymentFSMRepository (stub for TxHandle compliance)
// ============================================================

type mockPaymentFSMRepo struct{}

func (m *mockPaymentFSMRepo) GetForUpdate(ctx context.Context, organizationID, paymentID uuid.UUID) (*domain.PaymentForUpdate, error) {
	return nil, domain.ErrPaymentNotFound
}

func (m *mockPaymentFSMRepo) UpdateStatus(ctx context.Context, organizationID, paymentID uuid.UUID, newStatus domain.PaymentStatus) error {
	return nil
}

func (m *mockPaymentFSMRepo) InsertHistory(ctx context.Context, entry domain.TransitionHistory) error {
	return nil
}

func (m *mockPaymentFSMRepo) ListHistory(ctx context.Context, organizationID, paymentID uuid.UUID) ([]domain.TransitionHistory, error) {
	return nil, nil
}

// ============================================================
// Mock CredentialRepository
// ============================================================

type mockCredentialRepo struct {
	createFn     func(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error)
	getByIDFn    func(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error)
	listFn       func(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.CredentialRecord, error)
	softDeleteFn func(ctx context.Context, orgID, credentialID uuid.UUID) error
}

var _ domain.CredentialRepository = (*mockCredentialRepo)(nil)

func (m *mockCredentialRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateCredentialInput) (*domain.CredentialRecord, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return nil, errors.New("not implemented")
}

func (m *mockCredentialRepo) GetByID(ctx context.Context, orgID, credentialID uuid.UUID) (*domain.CredentialRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, credentialID)
	}
	return nil, domain.ErrCredentialNotFound
}

func (m *mockCredentialRepo) List(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.CredentialRecord, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, params)
	}
	return nil, nil
}

func (m *mockCredentialRepo) SoftDelete(ctx context.Context, orgID, credentialID uuid.UUID) error {
	if m.softDeleteFn != nil {
		return m.softDeleteFn(ctx, orgID, credentialID)
	}
	return nil
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

// ============================================================
// Mock CredentialLogRepository
// ============================================================

type mockCredentialLogRepo struct {
	listFn    func(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error)
	getByIDFn func(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error)
}

var _ domain.CredentialLogRepository = (*mockCredentialLogRepo)(nil)

func (m *mockCredentialLogRepo) List(ctx context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, input)
	}
	return nil, nil
}

func (m *mockCredentialLogRepo) GetByID(ctx context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, logID)
	}
	return nil, domain.ErrCredentialLogNotFound
}

// ============================================================
// Mock MemberRepository
// ============================================================

type mockMemberRepo struct {
	addMemberFn        func(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role, invitedBy *uuid.UUID) (*domain.OrganizationMember, error)
	getMemberFn        func(ctx context.Context, orgID, userAccountID uuid.UUID) (*domain.OrganizationMember, error)
	listMembersFn      func(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.OrganizationMember, int32, error)
	updateMemberRoleFn func(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role) (*domain.OrganizationMember, error)
	removeMemberFn     func(ctx context.Context, orgID, userAccountID uuid.UUID) error
	listOrgsForUserFn  func(ctx context.Context, userAccountID uuid.UUID) ([]domain.OrganizationRef, error)
}

var _ domain.MemberRepository = (*mockMemberRepo)(nil)

func (m *mockMemberRepo) AddMember(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role, invitedBy *uuid.UUID) (*domain.OrganizationMember, error) {
	if m.addMemberFn != nil {
		return m.addMemberFn(ctx, orgID, userAccountID, role, invitedBy)
	}
	return nil, errors.New("not implemented")
}

func (m *mockMemberRepo) GetMember(ctx context.Context, orgID, userAccountID uuid.UUID) (*domain.OrganizationMember, error) {
	if m.getMemberFn != nil {
		return m.getMemberFn(ctx, orgID, userAccountID)
	}
	return nil, domain.ErrNotFound
}

func (m *mockMemberRepo) ListMembers(ctx context.Context, orgID uuid.UUID, params domain.ListParams) ([]domain.OrganizationMember, int32, error) {
	if m.listMembersFn != nil {
		return m.listMembersFn(ctx, orgID, params)
	}
	return nil, 0, nil
}

func (m *mockMemberRepo) UpdateMemberRole(ctx context.Context, orgID, userAccountID uuid.UUID, role domain.Role) (*domain.OrganizationMember, error) {
	if m.updateMemberRoleFn != nil {
		return m.updateMemberRoleFn(ctx, orgID, userAccountID, role)
	}
	return nil, domain.ErrNotFound
}

func (m *mockMemberRepo) RemoveMember(ctx context.Context, orgID, userAccountID uuid.UUID) error {
	if m.removeMemberFn != nil {
		return m.removeMemberFn(ctx, orgID, userAccountID)
	}
	return nil
}

func (m *mockMemberRepo) ListOrganizationsForUser(ctx context.Context, userAccountID uuid.UUID) ([]domain.OrganizationRef, error) {
	if m.listOrgsForUserFn != nil {
		return m.listOrgsForUserFn(ctx, userAccountID)
	}
	return nil, nil
}

// ============================================================
// Mock PermissionRepository
// ============================================================

type mockPermissionRepo struct {
	upsertFn        func(ctx context.Context, memberID uuid.UUID, permission domain.Permission, granted bool) error
	deleteFn        func(ctx context.Context, memberID uuid.UUID, permission domain.Permission) error
	listForMemberFn func(ctx context.Context, memberID uuid.UUID) ([]domain.MemberPermission, error)
}

var _ domain.PermissionRepository = (*mockPermissionRepo)(nil)

func (m *mockPermissionRepo) Upsert(ctx context.Context, memberID uuid.UUID, permission domain.Permission, granted bool) error {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, memberID, permission, granted)
	}
	return nil
}

func (m *mockPermissionRepo) Delete(ctx context.Context, memberID uuid.UUID, permission domain.Permission) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, memberID, permission)
	}
	return nil
}

func (m *mockPermissionRepo) ListForMember(ctx context.Context, memberID uuid.UUID) ([]domain.MemberPermission, error) {
	if m.listForMemberFn != nil {
		return m.listForMemberFn(ctx, memberID)
	}
	return nil, nil
}
