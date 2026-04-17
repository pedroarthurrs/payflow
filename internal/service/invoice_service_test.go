package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ============================================================
// Mock InvoiceLogRepository
// ============================================================

type mockInvoiceLogRepo struct {
	createFn         func(ctx context.Context, organizationID uuid.UUID, input domain.CreateInvoiceLogInput) (*domain.InvoiceLog, error)
	listByInvoiceFn  func(ctx context.Context, organizationID, invoiceID uuid.UUID, params domain.ListParams) ([]domain.InvoiceLog, error)
	countByInvoiceFn func(ctx context.Context, organizationID, invoiceID uuid.UUID) (int64, error)
}

var _ domain.InvoiceLogRepository = (*mockInvoiceLogRepo)(nil)

func (m *mockInvoiceLogRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateInvoiceLogInput) (*domain.InvoiceLog, error) {
	if m.createFn != nil {
		return m.createFn(ctx, organizationID, input)
	}
	return &domain.InvoiceLog{ID: uuid.New(), OrganizationID: organizationID, InvoiceID: input.InvoiceID}, nil
}

func (m *mockInvoiceLogRepo) ListByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID, params domain.ListParams) ([]domain.InvoiceLog, error) {
	if m.listByInvoiceFn != nil {
		return m.listByInvoiceFn(ctx, organizationID, invoiceID, params)
	}
	return nil, nil
}

func (m *mockInvoiceLogRepo) CountByInvoice(ctx context.Context, organizationID, invoiceID uuid.UUID) (int64, error) {
	if m.countByInvoiceFn != nil {
		return m.countByInvoiceFn(ctx, organizationID, invoiceID)
	}
	return 0, nil
}

// ============================================================
// Mock InvoiceSettingRepository
// ============================================================

type mockInvoiceSettingRepo struct {
	upsertFn          func(ctx context.Context, organizationID uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error)
	getByOrgAndTypeFn func(ctx context.Context, organizationID uuid.UUID, invoiceType domain.InvoiceType) (*domain.InvoiceSetting, error)
	listFn            func(ctx context.Context, organizationID uuid.UUID) ([]domain.InvoiceSetting, error)
	deleteFn          func(ctx context.Context, organizationID, settingID uuid.UUID) error
}

var _ domain.InvoiceSettingRepository = (*mockInvoiceSettingRepo)(nil)

func (m *mockInvoiceSettingRepo) Upsert(ctx context.Context, organizationID uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error) {
	if m.upsertFn != nil {
		return m.upsertFn(ctx, organizationID, input)
	}
	return &domain.InvoiceSetting{ID: uuid.New(), OrganizationID: organizationID, Type: input.Type, IsActive: input.IsActive, IssueOn: input.IssueOn}, nil
}

func (m *mockInvoiceSettingRepo) GetByOrgAndType(ctx context.Context, organizationID uuid.UUID, invoiceType domain.InvoiceType) (*domain.InvoiceSetting, error) {
	if m.getByOrgAndTypeFn != nil {
		return m.getByOrgAndTypeFn(ctx, organizationID, invoiceType)
	}
	return nil, domain.ErrInvoiceSettingNotFound
}

func (m *mockInvoiceSettingRepo) List(ctx context.Context, organizationID uuid.UUID) ([]domain.InvoiceSetting, error) {
	if m.listFn != nil {
		return m.listFn(ctx, organizationID)
	}
	return nil, nil
}

func (m *mockInvoiceSettingRepo) Delete(ctx context.Context, organizationID, settingID uuid.UUID) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, organizationID, settingID)
	}
	return nil
}

// ============================================================
// Mock WebhookDispatcher
// ============================================================

type mockDispatcher struct {
	dispatched []string
}

func (m *mockDispatcher) Dispatch(_ context.Context, _ uuid.UUID, eventType, _ string, _ []byte) error {
	m.dispatched = append(m.dispatched, eventType)
	return nil
}

// ============================================================
// Helper: build InvoiceService with mocks
// ============================================================

type invoiceTestDeps struct {
	invoiceRepo  *mockInvoiceRepoStub
	itemRepo     *mockInvoiceItemRepoStub
	logRepo      *mockInvoiceLogRepo
	settingRepo  *mockInvoiceSettingRepo
	customerRepo *mockCustomerRepo
	subRepo      *mockSubscriptionRepoStub
	subItemRepo  *mockSubscriptionItemRepoStub
	planRepo     *mockPlanRepo
	txMgr        *mockTxManager
	dispatcher   *mockDispatcher
}

func newInvoiceTestDeps() *invoiceTestDeps {
	return &invoiceTestDeps{
		invoiceRepo:  &mockInvoiceRepoStub{},
		itemRepo:     &mockInvoiceItemRepoStub{},
		logRepo:      &mockInvoiceLogRepo{},
		settingRepo:  &mockInvoiceSettingRepo{},
		customerRepo: &mockCustomerRepo{},
		subRepo:      &mockSubscriptionRepoStub{},
		subItemRepo:  &mockSubscriptionItemRepoStub{},
		planRepo:     &mockPlanRepo{},
		txMgr:        &mockTxManager{},
		dispatcher:   &mockDispatcher{},
	}
}

func (d *invoiceTestDeps) service() *InvoiceService {
	return NewInvoiceService(
		d.invoiceRepo,
		d.itemRepo,
		d.logRepo,
		d.settingRepo,
		d.customerRepo,
		d.subRepo,
		d.subItemRepo,
		d.planRepo,
		d.txMgr,
		d.dispatcher,
	)
}

// ============================================================
// Service Tests: Create
// ============================================================

func TestInvoiceCreate_WithItems_HappyPath(t *testing.T) {
	deps := newInvoiceTestDeps()
	deps.customerRepo.getByIDFn = customerExistsMock()

	svc := deps.service()
	orgID := uuid.New()
	input := domain.CreateInvoiceInput{
		CustomerID:  uuid.New(),
		InvoiceType: domain.InvTypeNational,
		Items: []domain.CreateInvoiceItemInput{
			{Description: "Item A", Quantity: "2", UnitPriceCents: 1000, TotalPriceCents: 2000},
			{Description: "Item B", Quantity: "1", UnitPriceCents: 500, TotalPriceCents: 500},
		},
	}

	invoice, err := svc.Create(context.Background(), orgID, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoice == nil {
		t.Fatal("expected invoice, got nil")
	}
	// amount_cents should be sum of total_price_cents: 2000 + 500 = 2500
	if invoice.AmountCents != 2500 {
		t.Errorf("expected amount_cents=2500, got %d", invoice.AmountCents)
	}
	if len(invoice.Items) != 2 {
		t.Errorf("expected 2 items, got %d", len(invoice.Items))
	}
}

func TestInvoiceCreate_WithoutItems_ReturnsError(t *testing.T) {
	deps := newInvoiceTestDeps()
	svc := deps.service()

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateInvoiceInput{
		CustomerID:  uuid.New(),
		InvoiceType: domain.InvTypeNational,
		Items:       []domain.CreateInvoiceItemInput{},
	})
	if !errors.Is(err, domain.ErrInvoiceItemsRequired) {
		t.Errorf("expected ErrInvoiceItemsRequired, got %v", err)
	}
}

func TestInvoiceCreate_CustomerNotFound_ReturnsError(t *testing.T) {
	deps := newInvoiceTestDeps()
	deps.customerRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
		return nil, domain.ErrCustomerNotFound
	}
	svc := deps.service()

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateInvoiceInput{
		CustomerID:  uuid.New(),
		InvoiceType: domain.InvTypeNational,
		Items: []domain.CreateInvoiceItemInput{
			{Description: "Item", Quantity: "1", UnitPriceCents: 100, TotalPriceCents: 100},
		},
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Errorf("expected ErrCustomerNotFound, got %v", err)
	}
}

// ============================================================
// Service Tests: Transition
// ============================================================

func TestInvoiceTransition_ValidPendingToSent(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	invID := uuid.New()

	deps.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
		return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: domain.InvStatusPending}, nil
	}
	deps.invoiceRepo.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, newStatus domain.InvoiceStatus) (*domain.Invoice, error) {
		return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: newStatus}, nil
	}

	var logCreated bool
	deps.logRepo.createFn = func(_ context.Context, _ uuid.UUID, _ domain.CreateInvoiceLogInput) (*domain.InvoiceLog, error) {
		logCreated = true
		return &domain.InvoiceLog{ID: uuid.New()}, nil
	}

	svc := deps.service()
	invoice, err := svc.Transition(context.Background(), orgID, invID, domain.InvStatusSent, "sending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoice.Status != domain.InvStatusSent {
		t.Errorf("expected status=sent, got %s", invoice.Status)
	}
	if !logCreated {
		t.Error("expected invoice log to be created")
	}
}

func TestInvoiceTransition_InvalidPaidToSent(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	invID := uuid.New()

	deps.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
		return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: domain.InvStatusPaid}, nil
	}

	svc := deps.service()
	_, err := svc.Transition(context.Background(), orgID, invID, domain.InvStatusSent, "")
	if err == nil {
		t.Fatal("expected error for paid→sent transition")
	}
	if !errors.Is(err, domain.ErrTerminalState) {
		t.Errorf("expected ErrTerminalState, got %v", err)
	}
}

// ============================================================
// Service Tests: Cancel
// ============================================================

func TestInvoiceCancel_PendingInvoice(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	invID := uuid.New()

	deps.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
		return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: domain.InvStatusPending}, nil
	}
	deps.invoiceRepo.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, newStatus domain.InvoiceStatus) (*domain.Invoice, error) {
		return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: newStatus}, nil
	}

	svc := deps.service()
	invoice, err := svc.Cancel(context.Background(), orgID, invID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoice.Status != domain.InvStatusCanceled {
		t.Errorf("expected status=canceled, got %s", invoice.Status)
	}
}

func TestInvoiceCancel_AlreadyCanceled(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	invID := uuid.New()

	deps.invoiceRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Invoice, error) {
		return &domain.Invoice{ID: invID, OrganizationID: orgID, Status: domain.InvStatusCanceled}, nil
	}

	svc := deps.service()
	_, err := svc.Cancel(context.Background(), orgID, invID)
	// Cancel calls Transition(→canceled). FSM checks same-state first, so
	// canceled→canceled returns ErrSameStateTransition.
	if !errors.Is(err, domain.ErrSameStateTransition) {
		t.Errorf("expected ErrSameStateTransition, got %v", err)
	}
}

// ============================================================
// Service Tests: Settings CRUD
// ============================================================

func TestInvoiceUpsertSetting(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()

	callCount := 0
	deps.settingRepo.upsertFn = func(_ context.Context, _ uuid.UUID, input domain.UpsertInvoiceSettingInput) (*domain.InvoiceSetting, error) {
		callCount++
		return &domain.InvoiceSetting{
			ID:             uuid.New(),
			OrganizationID: orgID,
			Type:           input.Type,
			IsActive:       input.IsActive,
			IssueOn:        input.IssueOn,
		}, nil
	}

	svc := deps.service()

	// Create
	s1, err := svc.UpsertSetting(context.Background(), orgID, domain.UpsertInvoiceSettingInput{
		Type:     domain.InvTypeNational,
		IssueOn:  "payment_confirmed",
		IsActive: true,
	})
	if err != nil {
		t.Fatalf("unexpected error on create: %v", err)
	}
	if s1.Type != domain.InvTypeNational {
		t.Errorf("expected type=national, got %s", s1.Type)
	}

	// Update
	s2, err := svc.UpsertSetting(context.Background(), orgID, domain.UpsertInvoiceSettingInput{
		Type:     domain.InvTypeNational,
		IssueOn:  "manual",
		IsActive: false,
	})
	if err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}
	if s2.IsActive {
		t.Error("expected is_active=false after update")
	}
	if callCount != 2 {
		t.Errorf("expected 2 upsert calls, got %d", callCount)
	}
}

func TestInvoiceGetSettingByOrgAndType_Found(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	settingID := uuid.New()

	deps.settingRepo.getByOrgAndTypeFn = func(_ context.Context, _ uuid.UUID, _ domain.InvoiceType) (*domain.InvoiceSetting, error) {
		return &domain.InvoiceSetting{ID: settingID, OrganizationID: orgID, Type: domain.InvTypeNational}, nil
	}

	svc := deps.service()
	s, err := svc.GetSettingByOrgAndType(context.Background(), orgID, domain.InvTypeNational)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if s.ID != settingID {
		t.Errorf("expected setting ID %s, got %s", settingID, s.ID)
	}
}

func TestInvoiceGetSettingByOrgAndType_NotFound(t *testing.T) {
	deps := newInvoiceTestDeps()
	svc := deps.service()

	_, err := svc.GetSettingByOrgAndType(context.Background(), uuid.New(), domain.InvTypeNational)
	if !errors.Is(err, domain.ErrInvoiceSettingNotFound) {
		t.Errorf("expected ErrInvoiceSettingNotFound, got %v", err)
	}
}

func TestInvoiceListSettings(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()

	deps.settingRepo.listFn = func(_ context.Context, _ uuid.UUID) ([]domain.InvoiceSetting, error) {
		return []domain.InvoiceSetting{
			{ID: uuid.New(), Type: domain.InvTypeNational},
			{ID: uuid.New(), Type: domain.InvTypeInternational},
		}, nil
	}

	svc := deps.service()
	settings, err := svc.ListSettings(context.Background(), orgID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(settings) != 2 {
		t.Errorf("expected 2 settings, got %d", len(settings))
	}
}

func TestInvoiceDeleteSetting(t *testing.T) {
	deps := newInvoiceTestDeps()
	var deletedID uuid.UUID
	deps.settingRepo.deleteFn = func(_ context.Context, _, settingID uuid.UUID) error {
		deletedID = settingID
		return nil
	}

	svc := deps.service()
	settingID := uuid.New()
	err := svc.DeleteSetting(context.Background(), uuid.New(), settingID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedID != settingID {
		t.Errorf("expected deleted ID %s, got %s", settingID, deletedID)
	}
}

// ============================================================
// Service Tests: GenerateFromSubscription
// ============================================================

func TestInvoiceGenerateFromSubscription_HappyPath(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	subID := uuid.New()
	customerID := uuid.New()
	planID := uuid.New()

	deps.subRepo = &mockSubscriptionRepoStub{}
	// Override GetByID to return a subscription
	origSubRepo := deps.subRepo
	_ = origSubRepo

	deps.customerRepo.getByIDFn = customerExistsMock()

	// We need a custom sub repo that returns a subscription
	customSubRepo := &testSubRepo{
		sub: &domain.Subscription{ID: subID, OrganizationID: orgID, CustomerID: customerID},
	}
	customSubItemRepo := &testSubItemRepo{
		items: []domain.SubscriptionItem{
			{ID: uuid.New(), SubscriptionID: subID, PlanID: planID, Quantity: 2},
		},
	}

	deps.planRepo.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return &domain.Plan{ID: planID, Name: "Pro Plan", AmountCents: 1000}, nil
	}

	deps.settingRepo.getByOrgAndTypeFn = func(_ context.Context, _ uuid.UUID, invType domain.InvoiceType) (*domain.InvoiceSetting, error) {
		if invType == domain.InvTypeNational {
			return &domain.InvoiceSetting{ID: uuid.New(), Type: domain.InvTypeNational, IsActive: true, IssueOn: "payment_confirmed"}, nil
		}
		return nil, domain.ErrInvoiceSettingNotFound
	}

	svc := NewInvoiceService(
		deps.invoiceRepo, deps.itemRepo, deps.logRepo, deps.settingRepo,
		deps.customerRepo, customSubRepo, customSubItemRepo, deps.planRepo,
		deps.txMgr, deps.dispatcher,
	)

	invoice, err := svc.GenerateFromSubscription(context.Background(), orgID, subID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoice == nil {
		t.Fatal("expected invoice, got nil")
	}
	if invoice.AmountCents != 2000 {
		t.Errorf("expected amount_cents=2000 (2 * 1000), got %d", invoice.AmountCents)
	}
}

func TestInvoiceGenerateFromSubscription_NoSetting_Skip(t *testing.T) {
	deps := newInvoiceTestDeps()
	orgID := uuid.New()
	subID := uuid.New()

	customSubRepo := &testSubRepo{
		sub: &domain.Subscription{ID: subID, OrganizationID: orgID, CustomerID: uuid.New()},
	}
	customSubItemRepo := &testSubItemRepo{
		items: []domain.SubscriptionItem{
			{ID: uuid.New(), SubscriptionID: subID, PlanID: uuid.New(), Quantity: 1},
		},
	}

	// settingRepo default returns ErrInvoiceSettingNotFound for both types

	svc := NewInvoiceService(
		deps.invoiceRepo, deps.itemRepo, deps.logRepo, deps.settingRepo,
		deps.customerRepo, customSubRepo, customSubItemRepo, deps.planRepo,
		deps.txMgr, deps.dispatcher,
	)

	invoice, err := svc.GenerateFromSubscription(context.Background(), orgID, subID, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if invoice != nil {
		t.Error("expected nil invoice when no setting exists")
	}
}

// ── test-only subscription repo stubs ─────────────────────────────────────────

type testSubRepo struct {
	sub *domain.Subscription
}

func (r *testSubRepo) Create(_ context.Context, _ uuid.UUID, _ domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	return r.sub, nil
}
func (r *testSubRepo) GetByID(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
	if r.sub == nil {
		return nil, domain.ErrSubscriptionNotFound
	}
	return r.sub, nil
}
func (r *testSubRepo) List(_ context.Context, _ uuid.UUID, _ domain.SubscriptionListParams) ([]domain.Subscription, error) {
	return nil, nil
}
func (r *testSubRepo) Count(_ context.Context, _ uuid.UUID, _ domain.SubscriptionListParams) (int64, error) {
	return 0, nil
}
func (r *testSubRepo) UpdateStatus(_ context.Context, _, _ uuid.UUID, _ domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
	return nil, nil
}
func (r *testSubRepo) SetCancelAtPeriodEnd(_ context.Context, _, _ uuid.UUID, _ bool) (*domain.Subscription, error) {
	return nil, nil
}
func (r *testSubRepo) AdvancePeriod(_ context.Context, _, _ uuid.UUID, _, _ time.Time) (*domain.Subscription, error) {
	return nil, nil
}

type testSubItemRepo struct {
	items []domain.SubscriptionItem
}

func (r *testSubItemRepo) CreateItems(_ context.Context, _, _ uuid.UUID, _ []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	return nil, nil
}
func (r *testSubItemRepo) ReplaceItems(_ context.Context, _, _ uuid.UUID, _ []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	return nil, nil
}
func (r *testSubItemRepo) ListBySubscription(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
	return r.items, nil
}

// ============================================================
// FSM Tests
// ============================================================

func TestInvoiceFSM_AllValidTransitions(t *testing.T) {
	fsm := domain.NewInvoiceStateMachine()

	validTransitions := []struct {
		from, to domain.InvoiceStatus
	}{
		{domain.InvStatusPending, domain.InvStatusSent},
		{domain.InvStatusPending, domain.InvStatusCanceled},
		{domain.InvStatusSent, domain.InvStatusPaid},
		{domain.InvStatusSent, domain.InvStatusError},
		{domain.InvStatusSent, domain.InvStatusCanceled},
		{domain.InvStatusError, domain.InvStatusSent},
		{domain.InvStatusError, domain.InvStatusCanceled},
	}

	for _, tc := range validTransitions {
		err := fsm.Validate(tc.from, tc.to)
		if err != nil {
			t.Errorf("expected valid transition %s→%s, got error: %v", tc.from, tc.to, err)
		}
	}
}

func TestInvoiceFSM_InvalidTransitions(t *testing.T) {
	fsm := domain.NewInvoiceStateMachine()

	invalidTransitions := []struct {
		from, to domain.InvoiceStatus
	}{
		{domain.InvStatusPending, domain.InvStatusPaid},
		{domain.InvStatusPending, domain.InvStatusError},
		{domain.InvStatusSent, domain.InvStatusPending},
		{domain.InvStatusError, domain.InvStatusPaid},
		{domain.InvStatusError, domain.InvStatusPending},
	}

	for _, tc := range invalidTransitions {
		err := fsm.Validate(tc.from, tc.to)
		if err == nil {
			t.Errorf("expected error for invalid transition %s→%s", tc.from, tc.to)
		}
		if !errors.Is(err, domain.ErrInvalidInvoiceTransition) {
			t.Errorf("expected ErrInvalidInvoiceTransition for %s→%s, got %v", tc.from, tc.to, err)
		}
	}
}

func TestInvoiceFSM_SameStateTransition(t *testing.T) {
	fsm := domain.NewInvoiceStateMachine()

	allStatuses := []domain.InvoiceStatus{
		domain.InvStatusPending, domain.InvStatusSent, domain.InvStatusPaid,
		domain.InvStatusError, domain.InvStatusCanceled,
	}

	for _, s := range allStatuses {
		err := fsm.Validate(s, s)
		if !errors.Is(err, domain.ErrSameStateTransition) {
			t.Errorf("expected ErrSameStateTransition for %s→%s, got %v", s, s, err)
		}
	}
}

func TestInvoiceFSM_TransitionFromTerminalState(t *testing.T) {
	fsm := domain.NewInvoiceStateMachine()

	nonTerminal := []domain.InvoiceStatus{
		domain.InvStatusPending, domain.InvStatusSent, domain.InvStatusError,
	}

	// paid → any non-terminal should return ErrTerminalState
	for _, to := range nonTerminal {
		err := fsm.Validate(domain.InvStatusPaid, to)
		if !errors.Is(err, domain.ErrTerminalState) {
			t.Errorf("expected ErrTerminalState for paid→%s, got %v", to, err)
		}
	}

	// canceled → any non-terminal should return ErrInvoiceAlreadyCanceled
	for _, to := range nonTerminal {
		err := fsm.Validate(domain.InvStatusCanceled, to)
		if !errors.Is(err, domain.ErrInvoiceAlreadyCanceled) {
			t.Errorf("expected ErrInvoiceAlreadyCanceled for canceled→%s, got %v", to, err)
		}
	}
}
