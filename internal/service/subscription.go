package service

import (
	"context"
	"time"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// SubscriptionService handles subscription business logic.
type SubscriptionService struct {
	fsm       *domain.SubscriptionStateMachine
	fsmRepo   domain.SubscriptionFSMRepository
	subs      domain.SubscriptionRepository
	items     domain.SubscriptionItemRepository
	schedules domain.SubscriptionScheduleRepository
	history   domain.SubscriptionHistoryRepository
	plans     domain.PlanRepository
	customers domain.CustomerRepository
	txMgr     domain.TxManager
}

// NewSubscriptionService creates a SubscriptionService.
func NewSubscriptionService(
	subs domain.SubscriptionRepository,
	items domain.SubscriptionItemRepository,
	schedules domain.SubscriptionScheduleRepository,
	history domain.SubscriptionHistoryRepository,
	plans domain.PlanRepository,
	customers domain.CustomerRepository,
	txMgr domain.TxManager,
	fsmRepo domain.SubscriptionFSMRepository,
) *SubscriptionService {
	return &SubscriptionService{
		fsm:       domain.NewSubscriptionStateMachine(),
		fsmRepo:   fsmRepo,
		subs:      subs,
		items:     items,
		schedules: schedules,
		history:   history,
		plans:     plans,
		customers: customers,
		txMgr:     txMgr,
	}
}

// Create creates a subscription with its items atomically inside a transaction.
func (s *SubscriptionService) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	if len(input.Items) == 0 {
		return nil, domain.ErrNoSubscriptionItems
	}

	// Validate customer
	if _, err := s.customers.GetByID(ctx, organizationID, input.CustomerID); err != nil {
		return nil, err
	}

	// Validate all plans exist, are active, and belong to the org
	for _, item := range input.Items {
		plan, err := s.plans.GetByID(ctx, organizationID, item.PlanID)
		if err != nil {
			return nil, err
		}
		if !plan.Active {
			return nil, domain.ErrPlanNotFound
		}
	}

	// Determine initial status and period
	now := time.Now().UTC()
	input.CurrentPeriodStart = now
	input.Status = domain.SubStatusIncomplete
	if input.TrialEnd != nil && input.TrialEnd.After(now) {
		input.Status = domain.SubStatusTrialing
	}

	// Calculate period end from first item's plan
	firstPlan, _ := s.plans.GetByID(ctx, organizationID, input.Items[0].PlanID)
	input.CurrentPeriodEnd = calculatePeriodEnd(now, firstPlan.Interval, int(firstPlan.IntervalCount))

	var sub *domain.Subscription
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		subRepo := tx.SubscriptionRepo()
		itemRepo := tx.SubscriptionItemRepo()

		created, err := subRepo.Create(ctx, organizationID, input)
		if err != nil {
			return err
		}

		insertedItems, err := itemRepo.CreateItems(ctx, organizationID, created.ID, input.Items)
		if err != nil {
			return err
		}
		created.Items = insertedItems
		sub = created
		return nil
	})
	if err != nil {
		return nil, err
	}
	return sub, nil
}

// GetByID returns a subscription with its items populated.
func (s *SubscriptionService) GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	items, err := s.items.ListBySubscription(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	sub.Items = items
	return sub, nil
}

// List returns subscriptions with pagination.
func (s *SubscriptionService) List(ctx context.Context, organizationID uuid.UUID, params domain.SubscriptionListParams) ([]domain.Subscription, int64, error) {
	subs, err := s.subs.List(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.subs.Count(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	return subs, total, nil
}

// Transition atomically validates and applies a subscription status transition,
// recording a history entry within the same SQL transaction.
func (s *SubscriptionService) Transition(ctx context.Context, organizationID, subscriptionID uuid.UUID, toStatus domain.SubscriptionStatus, event domain.TransitionEvent) error {
	return s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		fsmRepo := tx.SubscriptionFSMRepo()

		current, err := fsmRepo.GetForUpdate(ctx, organizationID, subscriptionID)
		if err != nil {
			return err
		}

		if err := s.fsm.Validate(current.Status, toStatus); err != nil {
			return err
		}

		if err := fsmRepo.UpdateStatus(ctx, organizationID, subscriptionID, toStatus); err != nil {
			return err
		}

		oldStatus := current.Status
		return fsmRepo.InsertHistory(ctx, domain.SubscriptionStatusHistory{
			OrganizationID: organizationID,
			SubscriptionID: subscriptionID,
			OldStatus:      &oldStatus,
			NewStatus:      toStatus,
			EventSource:    event.Source,
			EventSourceID:  event.SourceID,
			Reason:         event.Reason,
		})
	})
}

// Cancel cancels a subscription immediately or at period end.
func (s *SubscriptionService) Cancel(ctx context.Context, organizationID, subscriptionID uuid.UUID, reason string, cancelAtPeriodEnd bool) (*domain.Subscription, error) {
	if cancelAtPeriodEnd {
		return s.subs.SetCancelAtPeriodEnd(ctx, organizationID, subscriptionID, true)
	}

	if err := s.Transition(ctx, organizationID, subscriptionID, domain.SubStatusCanceled, domain.TransitionEvent{
		Source: "api",
		Reason: reason,
	}); err != nil {
		return nil, err
	}
	return s.subs.GetByID(ctx, organizationID, subscriptionID)
}

// Pause pauses an active subscription.
func (s *SubscriptionService) Pause(ctx context.Context, organizationID, subscriptionID uuid.UUID, reason string) (*domain.Subscription, error) {
	if err := s.Transition(ctx, organizationID, subscriptionID, domain.SubStatusPaused, domain.TransitionEvent{
		Source: "api",
		Reason: reason,
	}); err != nil {
		return nil, err
	}
	return s.subs.GetByID(ctx, organizationID, subscriptionID)
}

// Resume resumes a paused subscription.
func (s *SubscriptionService) Resume(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	if err := s.Transition(ctx, organizationID, subscriptionID, domain.SubStatusActive, domain.TransitionEvent{
		Source: "api",
		Reason: "resumed",
	}); err != nil {
		return nil, err
	}
	return s.subs.GetByID(ctx, organizationID, subscriptionID)
}

// ReplaceItems replaces all items of a subscription atomically.
func (s *SubscriptionService) ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, itemInputs []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	if len(itemInputs) == 0 {
		return nil, domain.ErrNoSubscriptionItems
	}

	if _, err := s.subs.GetByID(ctx, organizationID, subscriptionID); err != nil {
		return nil, err
	}

	for _, item := range itemInputs {
		plan, err := s.plans.GetByID(ctx, organizationID, item.PlanID)
		if err != nil {
			return nil, err
		}
		if !plan.Active {
			return nil, domain.ErrPlanNotFound
		}
	}

	var insertedItems []domain.SubscriptionItem
	err := s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		itemRepo := tx.SubscriptionItemRepo()
		var err error
		insertedItems, err = itemRepo.ReplaceItems(ctx, organizationID, subscriptionID, itemInputs)
		return err
	})
	if err != nil {
		return nil, err
	}
	return insertedItems, nil
}

// ListItems returns items for a subscription.
func (s *SubscriptionService) ListItems(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionItem, error) {
	if _, err := s.subs.GetByID(ctx, organizationID, subscriptionID); err != nil {
		return nil, err
	}
	return s.items.ListBySubscription(ctx, organizationID, subscriptionID)
}

// ListHistory returns the status change history for a subscription with cursor pagination.
func (s *SubscriptionService) ListHistory(ctx context.Context, organizationID, subscriptionID uuid.UUID, limit int, cursor *domain.CursorToken) ([]domain.SubscriptionStatusHistory, *domain.CursorToken, error) {
	if _, err := s.subs.GetByID(ctx, organizationID, subscriptionID); err != nil {
		return nil, nil, err
	}
	return s.history.List(ctx, organizationID, subscriptionID, limit, cursor)
}

// CreateSchedule creates a future status or plan change schedule.
func (s *SubscriptionService) CreateSchedule(ctx context.Context, organizationID, subscriptionID uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
	sub, err := s.subs.GetByID(ctx, organizationID, subscriptionID)
	if err != nil {
		return nil, err
	}
	if s.fsm.IsTerminal(sub.Status) {
		return nil, domain.ErrSubscriptionCanceled
	}

	// Only one pending schedule at a time
	if _, err := s.schedules.GetPending(ctx, organizationID, subscriptionID); err == nil {
		return nil, domain.ErrScheduleAlreadyPending
	}

	input.SubscriptionID = subscriptionID
	return s.schedules.Create(ctx, organizationID, input)
}

// CancelSchedule cancels the pending schedule for a subscription.
func (s *SubscriptionService) CancelSchedule(ctx context.Context, organizationID, subscriptionID uuid.UUID) error {
	schedule, err := s.schedules.GetPending(ctx, organizationID, subscriptionID)
	if err != nil {
		return err
	}
	return s.schedules.Cancel(ctx, organizationID, schedule.ID)
}

// calculatePeriodEnd computes the end of the billing period from the start date.
func calculatePeriodEnd(start time.Time, interval string, intervalCount int) time.Time {
	switch interval {
	case "day":
		return start.AddDate(0, 0, intervalCount)
	case "week":
		return start.AddDate(0, 0, intervalCount*7)
	case "month":
		return start.AddDate(0, intervalCount, 0)
	case "year":
		return start.AddDate(intervalCount, 0, 0)
	default:
		return start.AddDate(0, 1, 0)
	}
}
