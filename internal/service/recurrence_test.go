package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── mock repos for recurrence tests ─────────────────────────────────────────

type mockSubscriptionRepo struct {
	getByIDFn           func(ctx context.Context, orgID, subID uuid.UUID) (*domain.Subscription, error)
	updateStatusFn      func(ctx context.Context, orgID, subID uuid.UUID, status domain.SubscriptionStatus, reason string) (*domain.Subscription, error)
	advancePeriodFn     func(ctx context.Context, orgID, subID uuid.UUID, newStart, newEnd time.Time) (*domain.Subscription, error)
	setCancelAtPeriodFn func(ctx context.Context, orgID, subID uuid.UUID, v bool) (*domain.Subscription, error)
	createFn            func(ctx context.Context, orgID uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error)
	listFn              func(ctx context.Context, orgID uuid.UUID, params domain.SubscriptionListParams) ([]domain.Subscription, error)
	countFn             func(ctx context.Context, orgID uuid.UUID, params domain.SubscriptionListParams) (int64, error)
}

var _ domain.SubscriptionRepository = (*mockSubscriptionRepo)(nil)

func (m *mockSubscriptionRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return nil, errors.New("not implemented")
}
func (m *mockSubscriptionRepo) GetByID(ctx context.Context, orgID, subID uuid.UUID) (*domain.Subscription, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, subID)
	}
	return nil, domain.ErrSubscriptionNotFound
}
func (m *mockSubscriptionRepo) List(ctx context.Context, orgID uuid.UUID, params domain.SubscriptionListParams) ([]domain.Subscription, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, params)
	}
	return nil, nil
}
func (m *mockSubscriptionRepo) Count(ctx context.Context, orgID uuid.UUID, params domain.SubscriptionListParams) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, orgID, params)
	}
	return 0, nil
}
func (m *mockSubscriptionRepo) UpdateStatus(ctx context.Context, orgID, subID uuid.UUID, status domain.SubscriptionStatus, reason string) (*domain.Subscription, error) {
	if m.updateStatusFn != nil {
		return m.updateStatusFn(ctx, orgID, subID, status, reason)
	}
	return &domain.Subscription{ID: subID, OrganizationID: orgID, Status: status}, nil
}
func (m *mockSubscriptionRepo) SetCancelAtPeriodEnd(ctx context.Context, orgID, subID uuid.UUID, v bool) (*domain.Subscription, error) {
	if m.setCancelAtPeriodFn != nil {
		return m.setCancelAtPeriodFn(ctx, orgID, subID, v)
	}
	return nil, nil
}
func (m *mockSubscriptionRepo) AdvancePeriod(ctx context.Context, orgID, subID uuid.UUID, newStart, newEnd time.Time) (*domain.Subscription, error) {
	if m.advancePeriodFn != nil {
		return m.advancePeriodFn(ctx, orgID, subID, newStart, newEnd)
	}
	return &domain.Subscription{ID: subID, OrganizationID: orgID, CurrentPeriodStart: newStart, CurrentPeriodEnd: newEnd}, nil
}

type mockSubscriptionItemRepo struct {
	listBySubFn func(ctx context.Context, orgID, subID uuid.UUID) ([]domain.SubscriptionItem, error)
	createFn    func(ctx context.Context, orgID, subID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error)
	replaceFn   func(ctx context.Context, orgID, subID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error)
}

var _ domain.SubscriptionItemRepository = (*mockSubscriptionItemRepo)(nil)

func (m *mockSubscriptionItemRepo) CreateItems(ctx context.Context, orgID, subID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, subID, items)
	}
	return nil, nil
}
func (m *mockSubscriptionItemRepo) ReplaceItems(ctx context.Context, orgID, subID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	if m.replaceFn != nil {
		return m.replaceFn(ctx, orgID, subID, items)
	}
	return nil, nil
}
func (m *mockSubscriptionItemRepo) ListBySubscription(ctx context.Context, orgID, subID uuid.UUID) ([]domain.SubscriptionItem, error) {
	if m.listBySubFn != nil {
		return m.listBySubFn(ctx, orgID, subID)
	}
	return nil, nil
}

type mockScheduleRepo struct {
	getPendingFn func(ctx context.Context, orgID, subID uuid.UUID) (*domain.SubscriptionSchedule, error)
	cancelFn     func(ctx context.Context, orgID, scheduleID uuid.UUID) error
	createFn     func(ctx context.Context, orgID uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error)
}

var _ domain.SubscriptionScheduleRepository = (*mockScheduleRepo)(nil)

func (m *mockScheduleRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return nil, nil
}
func (m *mockScheduleRepo) GetPending(ctx context.Context, orgID, subID uuid.UUID) (*domain.SubscriptionSchedule, error) {
	if m.getPendingFn != nil {
		return m.getPendingFn(ctx, orgID, subID)
	}
	return nil, domain.ErrScheduleNotFound
}
func (m *mockScheduleRepo) Cancel(ctx context.Context, orgID, scheduleID uuid.UUID) error {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, orgID, scheduleID)
	}
	return nil
}

type mockPlanRepo struct {
	getByIDFn func(ctx context.Context, orgID, planID uuid.UUID) (*domain.Plan, error)
}

func (m *mockPlanRepo) Create(_ context.Context, _ uuid.UUID, _ domain.PlanInput) (*domain.Plan, error) {
	return nil, nil
}
func (m *mockPlanRepo) GetByID(ctx context.Context, orgID, planID uuid.UUID) (*domain.Plan, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, planID)
	}
	return nil, domain.ErrPlanNotFound
}
func (m *mockPlanRepo) List(_ context.Context, _, _ uuid.UUID, _ domain.PlanListParams) ([]domain.Plan, error) {
	return nil, nil
}
func (m *mockPlanRepo) Count(_ context.Context, _, _ uuid.UUID, _ domain.PlanListParams) (int64, error) {
	return 0, nil
}
func (m *mockPlanRepo) Update(_ context.Context, _, _ uuid.UUID, _ domain.PlanUpdateInput) (*domain.Plan, error) {
	return nil, nil
}
func (m *mockPlanRepo) SoftDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockPlanRepo) HardDelete(_ context.Context, _, _ uuid.UUID) error { return nil }
func (m *mockPlanRepo) HasActiveSubscriptions(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return false, nil
}

type mockRecurrenceRepo struct {
	listDueFn func(ctx context.Context, now time.Time, batchSize int) ([]domain.SubscriptionBillingCandidate, error)
	lockFn    func(ctx context.Context, orgID, subID uuid.UUID, until time.Time) error
	unlockFn  func(ctx context.Context, orgID, subID uuid.UUID) error
	unlocked  []uuid.UUID
}

var _ domain.RecurrenceRepository = (*mockRecurrenceRepo)(nil)

func (m *mockRecurrenceRepo) ListDueForBilling(ctx context.Context, now time.Time, batchSize int) ([]domain.SubscriptionBillingCandidate, error) {
	if m.listDueFn != nil {
		return m.listDueFn(ctx, now, batchSize)
	}
	return nil, nil
}
func (m *mockRecurrenceRepo) LockForBilling(ctx context.Context, orgID, subID uuid.UUID, until time.Time) error {
	if m.lockFn != nil {
		return m.lockFn(ctx, orgID, subID, until)
	}
	return nil
}
func (m *mockRecurrenceRepo) UnlockBilling(ctx context.Context, orgID, subID uuid.UUID) error {
	m.unlocked = append(m.unlocked, subID)
	if m.unlockFn != nil {
		return m.unlockFn(ctx, orgID, subID)
	}
	return nil
}

type mockPaymentRepo struct {
	createFn           func(ctx context.Context, orgID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error)
	getByIDFn          func(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.Payment, error)
	updateGatewayRefFn func(ctx context.Context, orgID, paymentID uuid.UUID, name, ref string) error
	updateErrorLogFn   func(ctx context.Context, orgID, paymentID uuid.UUID, errLog json.RawMessage) error
	listFn             func(ctx context.Context, orgID uuid.UUID, params domain.PaymentListParams) ([]domain.Payment, error)
	countFn            func(ctx context.Context, orgID uuid.UUID, params domain.PaymentListParams) (int64, error)
	lastCreatedInput   *domain.PaymentInput
}

var _ domain.PaymentRepository = (*mockPaymentRepo)(nil)

func (m *mockPaymentRepo) Create(ctx context.Context, orgID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
	m.lastCreatedInput = &input
	if m.createFn != nil {
		return m.createFn(ctx, orgID, input)
	}
	return &domain.Payment{
		ID:             uuid.New(),
		OrganizationID: orgID,
		CustomerID:     input.CustomerID,
		AmountCents:    input.AmountCents,
		Status:         domain.StatusPending,
	}, nil
}
func (m *mockPaymentRepo) GetByID(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.Payment, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, orgID, paymentID)
	}
	return nil, domain.ErrPaymentNotFound
}
func (m *mockPaymentRepo) List(ctx context.Context, orgID uuid.UUID, params domain.PaymentListParams) ([]domain.Payment, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, params)
	}
	return nil, nil
}
func (m *mockPaymentRepo) Count(ctx context.Context, orgID uuid.UUID, params domain.PaymentListParams) (int64, error) {
	if m.countFn != nil {
		return m.countFn(ctx, orgID, params)
	}
	return 0, nil
}
func (m *mockPaymentRepo) UpdateGatewayReference(ctx context.Context, orgID, paymentID uuid.UUID, name, ref string) error {
	if m.updateGatewayRefFn != nil {
		return m.updateGatewayRefFn(ctx, orgID, paymentID, name, ref)
	}
	return nil
}
func (m *mockPaymentRepo) UpdateErrorLog(ctx context.Context, orgID, paymentID uuid.UUID, errLog json.RawMessage) error {
	if m.updateErrorLogFn != nil {
		return m.updateErrorLogFn(ctx, orgID, paymentID, errLog)
	}
	return nil
}

type mockGateway struct {
	chargeFn func(ctx context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error)
	refundFn func(ctx context.Context, req domain.GatewayRequest) (*domain.RefundResult, error)
	cancelFn func(ctx context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error)
}

var _ domain.Gateway = (*mockGateway)(nil)

func (m *mockGateway) Charge(ctx context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error) {
	if m.chargeFn != nil {
		return m.chargeFn(ctx, req)
	}
	return &domain.ChargeResult{GatewayName: "test", GatewayReference: "ref-123", Status: domain.StatusSucceeded}, nil
}
func (m *mockGateway) Refund(ctx context.Context, req domain.GatewayRequest) (*domain.RefundResult, error) {
	if m.refundFn != nil {
		return m.refundFn(ctx, req)
	}
	return &domain.RefundResult{GatewayReference: "ref-123", Status: domain.StatusRefunded}, nil
}
func (m *mockGateway) Cancel(ctx context.Context, req domain.GatewayRequest) (*domain.ChargeResult, error) {
	if m.cancelFn != nil {
		return m.cancelFn(ctx, req)
	}
	return &domain.ChargeResult{GatewayName: "test", GatewayReference: "ref-123", Status: domain.StatusCanceled}, nil
}

// ── helper to build RecurrenceService with mocks ─────────────────────────────

// recurrenceFSMRepo is a mock that succeeds for FSM transitions.
type recurrenceFSMRepo struct {
	status domain.PaymentStatus
}

func (m *recurrenceFSMRepo) GetForUpdate(_ context.Context, orgID, paymentID uuid.UUID) (*domain.PaymentForUpdate, error) {
	return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: m.status}, nil
}
func (m *recurrenceFSMRepo) UpdateStatus(_ context.Context, _, _ uuid.UUID, newStatus domain.PaymentStatus) error {
	m.status = newStatus
	return nil
}
func (m *recurrenceFSMRepo) InsertHistory(_ context.Context, _ domain.TransitionHistory) error {
	return nil
}
func (m *recurrenceFSMRepo) ListHistory(_ context.Context, _, _ uuid.UUID) ([]domain.TransitionHistory, error) {
	return nil, nil
}

type recurrenceTestHarness struct {
	subs      *mockSubscriptionRepo
	items     *mockSubscriptionItemRepo
	schedules *mockScheduleRepo
	plans     *mockPlanRepo
	recRepo   *mockRecurrenceRepo
	payments  *mockPaymentRepo
	gateway   *mockGateway
	txMgr     *mockTxManager
	fsmRepo   *recurrenceFSMRepo
	svc       *RecurrenceService
}

func newRecurrenceHarness() *recurrenceTestHarness {
	subs := &mockSubscriptionRepo{}
	items := &mockSubscriptionItemRepo{}
	schedules := &mockScheduleRepo{}
	plans := &mockPlanRepo{}
	recRepo := &mockRecurrenceRepo{}
	payments := &mockPaymentRepo{}
	gw := &mockGateway{}
	fsmRepo := &recurrenceFSMRepo{status: domain.StatusPending}

	txMgr := &mockTxManager{
		withTxFn: func(ctx context.Context, fn func(domain.TxHandle) error) error {
			return fn(&mockTxHandle{paymentFSM: fsmRepo})
		},
	}

	paymentSvc := NewPaymentService(payments, txMgr, fsmRepo, nil)
	gatewaySvc := NewGatewayService(paymentSvc, gw)

	svc := NewRecurrenceService(subs, items, schedules, plans, recRepo, payments, gatewaySvc, nil, txMgr, 4)

	return &recurrenceTestHarness{
		subs:      subs,
		items:     items,
		schedules: schedules,
		plans:     plans,
		recRepo:   recRepo,
		payments:  payments,
		gateway:   gw,
		txMgr:     txMgr,
		fsmRepo:   fsmRepo,
		svc:       svc,
	}
}

func makeActiveSub(orgID, subID, customerID uuid.UUID) *domain.Subscription {
	return &domain.Subscription{
		ID:                 subID,
		OrganizationID:     orgID,
		CustomerID:         customerID,
		Status:             domain.SubStatusActive,
		CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(-1 * time.Hour),
		PaymentMethod:      "credit_card",
	}
}

func makePlan(planID, orgID uuid.UUID, amountCents int32) *domain.Plan {
	return &domain.Plan{
		ID:             planID,
		OrganizationID: orgID,
		AmountCents:    amountCents,
		Currency:       "BRL",
		Interval:       "month",
		IntervalCount:  1,
		Active:         true,
	}
}

// ── Tests ────────────────────────────────────────────────────────────────────

func TestProcessBillingCycle_Success(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, o, s uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 2}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 5000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		return &domain.Payment{
			ID: paymentID, OrganizationID: orgID, CustomerID: customerID,
			AmountCents: input.AmountCents, Status: domain.StatusPending,
		}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{
			ID: paymentID, OrganizationID: orgID, CustomerID: customerID,
			AmountCents: 10000, Status: domain.StatusPending,
			PaymentMethod: &method, Currency: &currency,
		}, nil
	}

	var advancedStart, advancedEnd time.Time
	h.subs.advancePeriodFn = func(_ context.Context, _, _ uuid.UUID, start, end time.Time) (*domain.Subscription, error) {
		advancedStart = start
		advancedEnd = end
		return &domain.Subscription{ID: subID, CurrentPeriodStart: start, CurrentPeriodEnd: end}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	// Verify payment was created with correct amount (2 × 5000 = 10000)
	if h.payments.lastCreatedInput == nil {
		t.Fatal("expected payment to be created")
	}
	if h.payments.lastCreatedInput.AmountCents != 10000 {
		t.Errorf("expected amount 10000, got %d", h.payments.lastCreatedInput.AmountCents)
	}
	if h.payments.lastCreatedInput.SubscriptionID == nil || *h.payments.lastCreatedInput.SubscriptionID != subID {
		t.Error("expected subscription_id to be set on payment")
	}

	// Verify period was advanced
	if advancedStart.IsZero() || advancedEnd.IsZero() {
		t.Error("expected period to be advanced")
	}
	if !advancedEnd.After(advancedStart) {
		t.Error("expected new period end to be after start")
	}
}

func TestProcessBillingCycle_SkipTerminalState(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID := uuid.New(), uuid.New()

	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return &domain.Subscription{
			ID: subID, OrganizationID: orgID, Status: domain.SubStatusCanceled,
		}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected nil for terminal state, got %v", err)
	}

	// Should unlock since it's terminal
	if len(h.recRepo.unlocked) == 0 {
		t.Error("expected billing to be unlocked for terminal subscription")
	}
}

func TestProcessBillingCycle_CancelAtPeriodEnd(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID := uuid.New(), uuid.New()

	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return &domain.Subscription{
			ID:                subID,
			OrganizationID:    orgID,
			Status:            domain.SubStatusActive,
			CancelAtPeriodEnd: true,
			CurrentPeriodEnd:  time.Now().Add(-1 * time.Hour),
			PaymentMethod:     "credit_card",
		}, nil
	}

	var canceledStatus domain.SubscriptionStatus
	h.subs.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, reason string) (*domain.Subscription, error) {
		canceledStatus = status
		return &domain.Subscription{ID: subID, Status: status}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if canceledStatus != domain.SubStatusCanceled {
		t.Errorf("expected subscription to be canceled, got %s", canceledStatus)
	}
	if len(h.recRepo.unlocked) == 0 {
		t.Error("expected billing to be unlocked after cancel_at_period_end")
	}
}

func TestProcessBillingCycle_TrialToActive(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := &domain.Subscription{
		ID:                 subID,
		OrganizationID:     orgID,
		CustomerID:         customerID,
		Status:             domain.SubStatusTrialing,
		CurrentPeriodStart: time.Now().Add(-14 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(-1 * time.Hour),
		PaymentMethod:      "credit_card",
	}

	var transitionedToActive bool
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.subs.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
		if status == domain.SubStatusActive {
			transitionedToActive = true
			sub.Status = domain.SubStatusActive
		}
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 2000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		h.payments.lastCreatedInput = &input
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !transitionedToActive {
		t.Error("expected trialing subscription to transition to active")
	}
}

func TestProcessBillingCycle_ChargeFailure_ActiveToPastDue(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 5000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		h.payments.lastCreatedInput = &input
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 5000}, nil
	}

	// Gateway returns charge failed
	h.gateway.chargeFn = func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
		return nil, domain.ErrGatewayChargeFailed
	}

	var statusSet domain.SubscriptionStatus
	h.subs.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
		statusSet = status
		return &domain.Subscription{ID: subID, Status: status}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if !errors.Is(err, domain.ErrGatewayChargeFailed) {
		t.Fatalf("expected ErrGatewayChargeFailed, got %v", err)
	}

	if statusSet != domain.SubStatusPastDue {
		t.Errorf("expected subscription to move to past_due, got %s", statusSet)
	}

	// Should unlock on failure for retry
	if len(h.recRepo.unlocked) == 0 {
		t.Error("expected billing to be unlocked after charge failure")
	}
}

func TestProcessBillingCycle_PastDue_MaxRetries_ToUnpaid(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := &domain.Subscription{
		ID:                 subID,
		OrganizationID:     orgID,
		CustomerID:         customerID,
		Status:             domain.SubStatusPastDue,
		CurrentPeriodStart: time.Now().Add(-60 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(-1 * time.Hour),
		PaymentMethod:      "credit_card",
	}
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 3000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 3000}, nil
	}

	h.gateway.chargeFn = func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
		return nil, domain.ErrGatewayChargeFailed
	}

	var statusSet domain.SubscriptionStatus
	h.subs.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
		statusSet = status
		return &domain.Subscription{ID: subID, Status: status}, nil
	}

	// Attempt 4 = maxAttempts, should go to unpaid
	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 4)
	if !errors.Is(err, domain.ErrGatewayChargeFailed) {
		t.Fatalf("expected ErrGatewayChargeFailed, got %v", err)
	}

	if statusSet != domain.SubStatusUnpaid {
		t.Errorf("expected subscription to move to unpaid after max retries, got %s", statusSet)
	}
}

func TestProcessBillingCycle_ZeroAmount(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		plan := makePlan(planID, orgID, 0)
		return plan, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if !errors.Is(err, domain.ErrBillingAmountZero) {
		t.Fatalf("expected ErrBillingAmountZero, got %v", err)
	}

	// Should unlock since we're not charging
	if len(h.recRepo.unlocked) == 0 {
		t.Error("expected billing to be unlocked for zero amount")
	}
}

func TestProcessBillingCycle_SubscriptionNotFound(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID := uuid.New(), uuid.New()

	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return nil, domain.ErrSubscriptionNotFound
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if !errors.Is(err, domain.ErrSubscriptionNotFound) {
		t.Fatalf("expected ErrSubscriptionNotFound, got %v", err)
	}
}

func TestProcessBillingCycle_GatewayUnavailable_PropagatesError(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 5000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 5000}, nil
	}

	h.gateway.chargeFn = func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
		return nil, domain.ErrGatewayUnavailable
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err == nil {
		t.Fatal("expected error for gateway unavailable")
	}

	// Should unlock so retry can happen
	if len(h.recRepo.unlocked) == 0 {
		t.Error("expected billing to be unlocked after gateway unavailable")
	}
}

func TestProcessBillingCycle_MultipleItems_SumsCorrectly(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	plan1ID, plan2ID := uuid.New(), uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{
			{PlanID: plan1ID, Quantity: 3},
			{PlanID: plan2ID, Quantity: 2},
		}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, pID uuid.UUID) (*domain.Plan, error) {
		if pID == plan1ID {
			return makePlan(plan1ID, orgID, 1000), nil
		}
		return makePlan(plan2ID, orgID, 2500), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		h.payments.lastCreatedInput = &input
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 8000}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if h.payments.lastCreatedInput.AmountCents != 8000 {
		t.Errorf("expected total 8000, got %d", h.payments.lastCreatedInput.AmountCents)
	}
}

func TestProcessBillingCycle_IdempotencyKey_Format(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	periodStart := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	sub := &domain.Subscription{
		ID:                 subID,
		OrganizationID:     orgID,
		CustomerID:         customerID,
		Status:             domain.SubStatusActive,
		CurrentPeriodStart: periodStart,
		CurrentPeriodEnd:   time.Now().Add(-1 * time.Hour),
		PaymentMethod:      "credit_card",
	}
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 1000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		h.payments.lastCreatedInput = &input
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedKey := "sub_" + subID.String() + "_2026-03-15"
	if h.payments.lastCreatedInput.IdempotencyKey != expectedKey {
		t.Errorf("expected idempotency key %q, got %q", expectedKey, h.payments.lastCreatedInput.IdempotencyKey)
	}
}

func TestProcessBillingCycle_PastDue_SuccessRestoresActive(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	sub := &domain.Subscription{
		ID:                 subID,
		OrganizationID:     orgID,
		CustomerID:         customerID,
		Status:             domain.SubStatusPastDue,
		CurrentPeriodStart: time.Now().Add(-30 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(-1 * time.Hour),
		PaymentMethod:      "credit_card",
	}
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 5000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 5000}, nil
	}

	var restoredToActive bool
	h.subs.updateStatusFn = func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
		if status == domain.SubStatusActive {
			restoredToActive = true
		}
		return &domain.Subscription{ID: subID, Status: status}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 2)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !restoredToActive {
		t.Error("expected past_due subscription to be restored to active after successful charge")
	}
}

func TestProcessBillingCycle_ApplyPendingSchedule(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID, scheduleID := uuid.New(), uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 5000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 5000}, nil
	}

	var scheduleCanceled bool
	h.schedules.getPendingFn = func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionSchedule, error) {
		return &domain.SubscriptionSchedule{ID: scheduleID, SubscriptionID: subID}, nil
	}
	h.schedules.cancelFn = func(_ context.Context, _, sID uuid.UUID) error {
		if sID == scheduleID {
			scheduleCanceled = true
		}
		return nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !scheduleCanceled {
		t.Error("expected pending schedule to be applied (canceled) after successful billing")
	}
}

func TestProcessBillingCycle_CancelAtPeriodEnd_NotExpired_ContinuesBilling(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()
	planID := uuid.New()

	// Cancel is set but period hasn't ended yet (period_end in the future relative to sub)
	sub := &domain.Subscription{
		ID:                 subID,
		OrganizationID:     orgID,
		CustomerID:         customerID,
		Status:             domain.SubStatusActive,
		CancelAtPeriodEnd:  true,
		CurrentPeriodStart: time.Now().Add(-15 * 24 * time.Hour),
		CurrentPeriodEnd:   time.Now().Add(15 * 24 * time.Hour), // still in the future
		PaymentMethod:      "credit_card",
	}
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{{PlanID: planID, Quantity: 1}}, nil
	}
	h.plans.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
		return makePlan(planID, orgID, 5000), nil
	}

	paymentID := uuid.New()
	h.payments.createFn = func(_ context.Context, _ uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, AmountCents: input.AmountCents}, nil
	}
	h.payments.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
		method := "credit_card"
		currency := "BRL"
		return &domain.Payment{ID: paymentID, Status: domain.StatusPending, PaymentMethod: &method, Currency: &currency, AmountCents: 5000}, nil
	}

	// It should still charge normally because the period hasn't ended
	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestProcessBillingCycle_EmptyItems(t *testing.T) {
	h := newRecurrenceHarness()
	orgID, subID, customerID := uuid.New(), uuid.New(), uuid.New()

	sub := makeActiveSub(orgID, subID, customerID)
	h.subs.getByIDFn = func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
		return sub, nil
	}
	h.items.listBySubFn = func(_ context.Context, _, _ uuid.UUID) ([]domain.SubscriptionItem, error) {
		return []domain.SubscriptionItem{}, nil
	}

	err := h.svc.ProcessBillingCycle(context.Background(), orgID, subID, 1)
	if !errors.Is(err, domain.ErrBillingAmountZero) {
		t.Fatalf("expected ErrBillingAmountZero for empty items, got %v", err)
	}
}
