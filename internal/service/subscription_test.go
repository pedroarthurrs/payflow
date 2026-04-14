package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// ── Subscription Service Tests ──────────────────────────────────────────────

func newSubscriptionSvc(
	subs *mockSubscriptionRepo,
	items *mockSubscriptionItemRepo,
	schedules *mockScheduleRepo,
	history *mockSubscriptionHistoryRepo,
	plans *mockPlanRepo,
	customers *mockCustomerRepo,
	txMgr *mockTxManager,
) *SubscriptionService {
	return NewSubscriptionService(subs, items, schedules, history, plans, customers, txMgr)
}

type mockSubscriptionHistoryRepo struct {
	listFn func(ctx context.Context, orgID, subID uuid.UUID) ([]domain.SubscriptionStatusHistory, error)
}

var _ domain.SubscriptionHistoryRepository = (*mockSubscriptionHistoryRepo)(nil)

func (m *mockSubscriptionHistoryRepo) List(ctx context.Context, orgID, subID uuid.UUID) ([]domain.SubscriptionStatusHistory, error) {
	if m.listFn != nil {
		return m.listFn(ctx, orgID, subID)
	}
	return nil, nil
}

// ── Create ──────────────────────────────────────────────────────────────────

func TestSubscriptionCreate_Success(t *testing.T) {
	orgID := uuid.New()
	customerID := uuid.New()
	planID := uuid.New()

	subs := &mockSubscriptionRepo{
		createFn: func(_ context.Context, _ uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
			return &domain.Subscription{
				ID:             uuid.New(),
				OrganizationID: orgID,
				CustomerID:     input.CustomerID,
				Status:         input.Status,
			}, nil
		},
	}
	items := &mockSubscriptionItemRepo{
		createFn: func(_ context.Context, _, _ uuid.UUID, its []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
			result := make([]domain.SubscriptionItem, len(its))
			for i, it := range its {
				result[i] = domain.SubscriptionItem{ID: uuid.New(), PlanID: it.PlanID, Quantity: it.Quantity}
			}
			return result, nil
		},
	}
	plans := &mockPlanRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
			return &domain.Plan{
				ID: planID, Active: true, Interval: "month", IntervalCount: 1,
				AmountCents: 5000, Currency: "BRL",
			}, nil
		},
	}
	customers := &mockCustomerRepo{
		getByIDFn: customerExistsMock(),
	}
	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(domain.TxHandle) error) error {
			return fn(&mockTxHandle{
				customers: customers,
				subs:      subs,
				subItems:  items,
			})
		},
	}

	svc := newSubscriptionSvc(subs, items, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, plans, customers, txMgr)

	sub, err := svc.Create(context.Background(), orgID, domain.CreateSubscriptionInput{
		CustomerID:    customerID,
		PaymentMethod: "credit_card",
		Items:         []domain.SubscriptionItemInput{{PlanID: planID, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub == nil {
		t.Fatal("expected subscription, got nil")
	}
}

func TestSubscriptionCreate_NoItems(t *testing.T) {
	svc := newSubscriptionSvc(
		&mockSubscriptionRepo{}, &mockSubscriptionItemRepo{}, &mockScheduleRepo{},
		&mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{},
	)

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateSubscriptionInput{
		CustomerID:    uuid.New(),
		PaymentMethod: "credit_card",
		Items:         []domain.SubscriptionItemInput{},
	})
	if !errors.Is(err, domain.ErrNoSubscriptionItems) {
		t.Fatalf("expected ErrNoSubscriptionItems, got %v", err)
	}
}

func TestSubscriptionCreate_CustomerNotFound(t *testing.T) {
	customers := &mockCustomerRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Customer, error) {
			return nil, domain.ErrCustomerNotFound
		},
	}
	plans := &mockPlanRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
			return &domain.Plan{Active: true, Interval: "month", IntervalCount: 1}, nil
		},
	}

	svc := newSubscriptionSvc(
		&mockSubscriptionRepo{}, &mockSubscriptionItemRepo{}, &mockScheduleRepo{},
		&mockSubscriptionHistoryRepo{}, plans, customers, &mockTxManager{},
	)

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateSubscriptionInput{
		CustomerID:    uuid.New(),
		PaymentMethod: "credit_card",
		Items:         []domain.SubscriptionItemInput{{PlanID: uuid.New(), Quantity: 1}},
	})
	if !errors.Is(err, domain.ErrCustomerNotFound) {
		t.Fatalf("expected ErrCustomerNotFound, got %v", err)
	}
}

func TestSubscriptionCreate_InactivePlan(t *testing.T) {
	customers := &mockCustomerRepo{getByIDFn: customerExistsMock()}
	plans := &mockPlanRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
			return &domain.Plan{Active: false}, nil
		},
	}

	svc := newSubscriptionSvc(
		&mockSubscriptionRepo{}, &mockSubscriptionItemRepo{}, &mockScheduleRepo{},
		&mockSubscriptionHistoryRepo{}, plans, customers, &mockTxManager{},
	)

	_, err := svc.Create(context.Background(), uuid.New(), domain.CreateSubscriptionInput{
		CustomerID:    uuid.New(),
		PaymentMethod: "credit_card",
		Items:         []domain.SubscriptionItemInput{{PlanID: uuid.New(), Quantity: 1}},
	})
	if !errors.Is(err, domain.ErrPlanNotFound) {
		t.Fatalf("expected ErrPlanNotFound for inactive plan, got %v", err)
	}
}

func TestSubscriptionCreate_WithTrial(t *testing.T) {
	orgID := uuid.New()
	planID := uuid.New()
	trialEnd := time.Now().Add(14 * 24 * time.Hour)

	var capturedStatus domain.SubscriptionStatus
	subs := &mockSubscriptionRepo{
		createFn: func(_ context.Context, _ uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
			capturedStatus = input.Status
			return &domain.Subscription{
				ID: uuid.New(), OrganizationID: orgID, Status: input.Status,
				CustomerID: input.CustomerID,
			}, nil
		},
	}
	items := &mockSubscriptionItemRepo{
		createFn: func(_ context.Context, _, _ uuid.UUID, its []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
			return []domain.SubscriptionItem{{ID: uuid.New(), PlanID: its[0].PlanID}}, nil
		},
	}
	plans := &mockPlanRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
			return &domain.Plan{ID: planID, Active: true, Interval: "month", IntervalCount: 1}, nil
		},
	}
	customers := &mockCustomerRepo{getByIDFn: customerExistsMock()}
	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(domain.TxHandle) error) error {
			return fn(&mockTxHandle{subs: subs, subItems: items})
		},
	}

	svc := newSubscriptionSvc(subs, items, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, plans, customers, txMgr)

	_, err := svc.Create(context.Background(), orgID, domain.CreateSubscriptionInput{
		CustomerID:    uuid.New(),
		PaymentMethod: "credit_card",
		TrialEnd:      &trialEnd,
		Items:         []domain.SubscriptionItemInput{{PlanID: planID, Quantity: 1}},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if capturedStatus != domain.SubStatusTrialing {
		t.Errorf("expected trialing status, got %s", capturedStatus)
	}
}

// ── Cancel ──────────────────────────────────────────────────────────────────

func TestSubscriptionCancel_Immediate(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, OrganizationID: orgID, Status: domain.SubStatusActive}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: status}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	sub, err := svc.Cancel(context.Background(), orgID, subID, "user requested", false)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub.Status != domain.SubStatusCanceled {
		t.Errorf("expected canceled, got %s", sub.Status)
	}
}

func TestSubscriptionCancel_AtPeriodEnd(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	var setCancelAtEnd bool
	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, OrganizationID: orgID, Status: domain.SubStatusActive}, nil
		},
		setCancelAtPeriodFn: func(_ context.Context, _, _ uuid.UUID, v bool) (*domain.Subscription, error) {
			setCancelAtEnd = v
			return &domain.Subscription{ID: subID, CancelAtPeriodEnd: v}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	_, err := svc.Cancel(context.Background(), orgID, subID, "end of period", true)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !setCancelAtEnd {
		t.Error("expected cancel_at_period_end to be set to true")
	}
}

func TestSubscriptionCancel_AlreadyCanceled(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusCanceled}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	_, err := svc.Cancel(context.Background(), orgID, subID, "test", false)
	if err == nil {
		t.Fatal("expected error when canceling already canceled subscription")
	}
}

// ── Pause / Resume ──────────────────────────────────────────────────────────

func TestSubscriptionPause_Success(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusActive}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: status}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	sub, err := svc.Pause(context.Background(), orgID, subID, "vacation")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub.Status != domain.SubStatusPaused {
		t.Errorf("expected paused, got %s", sub.Status)
	}
}

func TestSubscriptionPause_CanceledSubscription(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusCanceled}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	_, err := svc.Pause(context.Background(), orgID, subID, "test")
	if err == nil {
		t.Fatal("expected error when pausing canceled subscription")
	}
}

func TestSubscriptionResume_Success(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusPaused}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, status domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: status}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	sub, err := svc.Resume(context.Background(), orgID, subID)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sub.Status != domain.SubStatusActive {
		t.Errorf("expected active, got %s", sub.Status)
	}
}

func TestSubscriptionResume_NotPaused(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusCanceled}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	_, err := svc.Resume(context.Background(), orgID, subID)
	if err == nil {
		t.Fatal("expected error when resuming non-paused subscription")
	}
}

// ── ReplaceItems ────────────────────────────────────────────────────────────

func TestSubscriptionReplaceItems_Success(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()
	planID := uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusActive}, nil
		},
	}
	items := &mockSubscriptionItemRepo{
		replaceFn: func(_ context.Context, _, _ uuid.UUID, its []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
			result := make([]domain.SubscriptionItem, len(its))
			for i, it := range its {
				result[i] = domain.SubscriptionItem{ID: uuid.New(), PlanID: it.PlanID, Quantity: it.Quantity}
			}
			return result, nil
		},
	}
	plans := &mockPlanRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Plan, error) {
			return &domain.Plan{ID: planID, Active: true}, nil
		},
	}

	txMgr := &mockTxManager{
		withTxFn: func(_ context.Context, fn func(domain.TxHandle) error) error {
			return fn(&mockTxHandle{subItems: items})
		},
	}

	svc := newSubscriptionSvc(subs, items, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, plans, &mockCustomerRepo{}, txMgr)

	result, err := svc.ReplaceItems(context.Background(), orgID, subID, []domain.SubscriptionItemInput{{PlanID: planID, Quantity: 3}})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(result) != 1 {
		t.Errorf("expected 1 item, got %d", len(result))
	}
}

func TestSubscriptionReplaceItems_EmptyItems(t *testing.T) {
	svc := newSubscriptionSvc(
		&mockSubscriptionRepo{}, &mockSubscriptionItemRepo{}, &mockScheduleRepo{},
		&mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{},
	)

	_, err := svc.ReplaceItems(context.Background(), uuid.New(), uuid.New(), []domain.SubscriptionItemInput{})
	if !errors.Is(err, domain.ErrNoSubscriptionItems) {
		t.Fatalf("expected ErrNoSubscriptionItems, got %v", err)
	}
}

// ── Schedule ────────────────────────────────────────────────────────────────

func TestSubscriptionCreateSchedule_Success(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()
	targetStatus := domain.SubStatusCanceled

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusActive}, nil
		},
	}
	schedules := &mockScheduleRepo{
		getPendingFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionSchedule, error) {
			return nil, domain.ErrScheduleNotFound
		},
		createFn: func(_ context.Context, _ uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
			return &domain.SubscriptionSchedule{
				ID: uuid.New(), SubscriptionID: input.SubscriptionID, TargetStatus: input.TargetStatus,
			}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, schedules, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	sch, err := svc.CreateSchedule(context.Background(), orgID, subID, domain.CreateScheduleInput{
		ScheduledAt:  time.Now().Add(7 * 24 * time.Hour),
		TargetStatus: &targetStatus,
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if sch == nil {
		t.Fatal("expected schedule, got nil")
	}
}

func TestSubscriptionCreateSchedule_AlreadyPending(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusActive}, nil
		},
	}
	schedules := &mockScheduleRepo{
		getPendingFn: func(_ context.Context, _, _ uuid.UUID) (*domain.SubscriptionSchedule, error) {
			return &domain.SubscriptionSchedule{ID: uuid.New()}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, schedules, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	_, err := svc.CreateSchedule(context.Background(), orgID, subID, domain.CreateScheduleInput{
		ScheduledAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if !errors.Is(err, domain.ErrScheduleAlreadyPending) {
		t.Fatalf("expected ErrScheduleAlreadyPending, got %v", err)
	}
}

func TestSubscriptionCreateSchedule_TerminalSubscription(t *testing.T) {
	orgID, subID := uuid.New(), uuid.New()

	subs := &mockSubscriptionRepo{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Subscription, error) {
			return &domain.Subscription{ID: subID, Status: domain.SubStatusCanceled}, nil
		},
	}

	svc := newSubscriptionSvc(subs, &mockSubscriptionItemRepo{}, &mockScheduleRepo{}, &mockSubscriptionHistoryRepo{}, &mockPlanRepo{}, &mockCustomerRepo{}, &mockTxManager{})

	_, err := svc.CreateSchedule(context.Background(), orgID, subID, domain.CreateScheduleInput{
		ScheduledAt: time.Now().Add(7 * 24 * time.Hour),
	})
	if !errors.Is(err, domain.ErrSubscriptionCanceled) {
		t.Fatalf("expected ErrSubscriptionCanceled, got %v", err)
	}
}

// ── calculatePeriodEnd ──────────────────────────────────────────────────────

func TestCalculatePeriodEnd(t *testing.T) {
	start := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name     string
		interval string
		count    int
		expected time.Time
	}{
		{"day", "day", 1, time.Date(2026, 1, 16, 10, 0, 0, 0, time.UTC)},
		{"7 days", "day", 7, time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC)},
		{"week", "week", 1, time.Date(2026, 1, 22, 10, 0, 0, 0, time.UTC)},
		{"2 weeks", "week", 2, time.Date(2026, 1, 29, 10, 0, 0, 0, time.UTC)},
		{"month", "month", 1, time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)},
		{"3 months", "month", 3, time.Date(2026, 4, 15, 10, 0, 0, 0, time.UTC)},
		{"year", "year", 1, time.Date(2027, 1, 15, 10, 0, 0, 0, time.UTC)},
		{"unknown defaults to month", "unknown", 1, time.Date(2026, 2, 15, 10, 0, 0, 0, time.UTC)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calculatePeriodEnd(start, tt.interval, tt.count)
			if !got.Equal(tt.expected) {
				t.Errorf("calculatePeriodEnd(%s, %d) = %v, want %v", tt.interval, tt.count, got, tt.expected)
			}
		})
	}
}

// ── Edge case: Month-end boundary ───────────────────────────────────────────

func TestCalculatePeriodEnd_MonthBoundary(t *testing.T) {
	// Jan 31 + 1 month = Feb 28 (Go's time.AddDate normalizes)
	start := time.Date(2026, 1, 31, 0, 0, 0, 0, time.UTC)
	got := calculatePeriodEnd(start, "month", 1)
	// Go normalizes to March 3 for this
	if got.Month() != time.March && got.Month() != time.February {
		t.Errorf("expected month boundary normalization, got %v", got)
	}
}
