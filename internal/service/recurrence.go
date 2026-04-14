package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// RecurrenceService handles the recurring billing cycle for subscriptions.
type RecurrenceService struct {
	subs        domain.SubscriptionRepository
	items       domain.SubscriptionItemRepository
	schedules   domain.SubscriptionScheduleRepository
	plans       domain.PlanRepository
	recRepo     domain.RecurrenceRepository
	payments    domain.PaymentRepository
	gatewaySvc  *GatewayService
	subFSM      *domain.SubscriptionStateMachine
	txMgr       domain.TxManager
	maxAttempts int
}

// NewRecurrenceService creates a RecurrenceService.
func NewRecurrenceService(
	subs domain.SubscriptionRepository,
	items domain.SubscriptionItemRepository,
	schedules domain.SubscriptionScheduleRepository,
	plans domain.PlanRepository,
	recRepo domain.RecurrenceRepository,
	payments domain.PaymentRepository,
	gatewaySvc *GatewayService,
	txMgr domain.TxManager,
	maxAttempts int,
) *RecurrenceService {
	return &RecurrenceService{
		subs:        subs,
		items:       items,
		schedules:   schedules,
		plans:       plans,
		recRepo:     recRepo,
		payments:    payments,
		gatewaySvc:  gatewaySvc,
		subFSM:      domain.NewSubscriptionStateMachine(),
		txMgr:       txMgr,
		maxAttempts: maxAttempts,
	}
}

// ProcessBillingCycle runs a full billing cycle for a single subscription.
// attemptNum is 1-based (first attempt = 1).
func (s *RecurrenceService) ProcessBillingCycle(ctx context.Context, orgID, subID uuid.UUID, attemptNum int) error {
	now := time.Now().UTC()

	// Step 1: Load subscription.
	sub, err := s.subs.GetByID(ctx, orgID, subID)
	if err != nil {
		return err
	}

	// Step 2: Skip terminal states.
	if s.subFSM.IsTerminal(sub.Status) {
		slog.Info("skipping terminal subscription", "subscription_id", subID, "status", sub.Status)
		_ = s.recRepo.UnlockBilling(ctx, orgID, subID)
		return nil
	}

	// Step 3: Cancel if cancel_at_period_end and period has ended.
	if sub.CancelAtPeriodEnd && !sub.CurrentPeriodEnd.After(now) {
		if _, err := s.subs.UpdateStatus(ctx, orgID, subID, domain.SubStatusCanceled, "cancel_at_period_end"); err != nil {
			return fmt.Errorf("canceling subscription at period end: %w", err)
		}
		_ = s.recRepo.UnlockBilling(ctx, orgID, subID)
		return nil
	}

	// Step 4: Transition trial → active when trial period has ended.
	if sub.Status == domain.SubStatusTrialing && !sub.CurrentPeriodEnd.After(now) {
		updated, err := s.subs.UpdateStatus(ctx, orgID, subID, domain.SubStatusActive, "trial_ended")
		if err != nil {
			return fmt.Errorf("transitioning trial to active: %w", err)
		}
		sub = updated
	}

	// Step 5: Load items and plans to compute billing amount.
	subItems, err := s.items.ListBySubscription(ctx, orgID, subID)
	if err != nil {
		return err
	}

	var totalAmountCents int32
	var currency string
	var billingInterval string
	var billingIntervalCount int

	for i, item := range subItems {
		plan, err := s.plans.GetByID(ctx, orgID, item.PlanID)
		if err != nil {
			return fmt.Errorf("loading plan %s: %w", item.PlanID, err)
		}
		totalAmountCents += plan.AmountCents * item.Quantity
		if i == 0 {
			currency = plan.Currency
			billingInterval = plan.Interval
			billingIntervalCount = int(plan.IntervalCount)
		}
	}

	// Step 6: Billing amount must be positive.
	if totalAmountCents <= 0 {
		slog.Warn("billing amount is zero, skipping", "subscription_id", subID)
		_ = s.recRepo.UnlockBilling(ctx, orgID, subID)
		return domain.ErrBillingAmountZero
	}

	// Step 7: Idempotency key based on subscription + period start.
	idempotencyKey := fmt.Sprintf("sub_%s_%s", subID, sub.CurrentPeriodStart.Format("2006-01-02"))

	// Step 8: Create payment record.
	payment, err := s.payments.Create(ctx, orgID, domain.PaymentInput{
		CustomerID:     sub.CustomerID,
		SubscriptionID: &subID,
		PaymentMethod:  sub.PaymentMethod,
		AmountCents:    totalAmountCents,
		Currency:       currency,
		IdempotencyKey: idempotencyKey,
	})
	if err != nil {
		return fmt.Errorf("creating payment: %w", err)
	}

	// Remember the subscription status before the charge for post-charge decisions.
	statusBeforeCharge := sub.Status

	// Step 9: Charge via gateway (handles FSM transitions internally).
	charged, chargeErr := s.gatewaySvc.Charge(ctx, orgID, payment.ID, "recurring_billing")

	// Step 10: Success path.
	if chargeErr == nil {
		slog.Info("recurring charge succeeded",
			"subscription_id", subID,
			"payment_id", charged.ID,
			"amount_cents", totalAmountCents)

		// If subscription was past_due, restore to active.
		if statusBeforeCharge == domain.SubStatusPastDue {
			if _, err := s.subs.UpdateStatus(ctx, orgID, subID, domain.SubStatusActive, "payment_succeeded"); err != nil {
				slog.Error("failed to restore subscription to active", "subscription_id", subID, "error", err)
			}
		}

		// Advance period (also clears billing_locked_until).
		newPeriodStart := sub.CurrentPeriodEnd
		newPeriodEnd := calculatePeriodEnd(newPeriodStart, billingInterval, billingIntervalCount)
		if _, err := s.subs.AdvancePeriod(ctx, orgID, subID, newPeriodStart, newPeriodEnd); err != nil {
			slog.Error("failed to advance subscription period", "subscription_id", subID, "error", err)
			return fmt.Errorf("advancing subscription period: %w", err)
		}

		// Apply any pending schedule (mark as applied).
		s.applyPendingSchedule(ctx, orgID, subID)
		return nil
	}

	// Step 11: Failure path.
	if errors.Is(chargeErr, domain.ErrGatewayChargeFailed) {
		slog.Warn("recurring charge failed",
			"subscription_id", subID,
			"payment_id", payment.ID,
			"attempt", attemptNum,
			"error", chargeErr)

		// Update subscription status based on failure.
		s.handleChargeFailure(ctx, orgID, subID, statusBeforeCharge, attemptNum)

		// Unlock so the subscription can be retried.
		_ = s.recRepo.UnlockBilling(ctx, orgID, subID)
		return chargeErr
	}

	// Gateway unavailable or unexpected error — unlock and propagate.
	_ = s.recRepo.UnlockBilling(ctx, orgID, subID)
	return fmt.Errorf("gateway error: %w", chargeErr)
}

// handleChargeFailure updates the subscription status based on the failure context.
func (s *RecurrenceService) handleChargeFailure(ctx context.Context, orgID, subID uuid.UUID, currentStatus domain.SubscriptionStatus, attemptNum int) {
	switch currentStatus {
	case domain.SubStatusActive:
		if _, err := s.subs.UpdateStatus(ctx, orgID, subID, domain.SubStatusPastDue, "payment_failed"); err != nil {
			slog.Error("failed to set subscription to past_due", "subscription_id", subID, "error", err)
		}
	case domain.SubStatusPastDue:
		if attemptNum >= s.maxAttempts {
			if _, err := s.subs.UpdateStatus(ctx, orgID, subID, domain.SubStatusUnpaid, "max_retries_exceeded"); err != nil {
				slog.Error("failed to set subscription to unpaid", "subscription_id", subID, "error", err)
			}
		}
	case domain.SubStatusTrialing, domain.SubStatusUnpaid,
		domain.SubStatusCanceled, domain.SubStatusPaused, domain.SubStatusIncomplete:
		slog.Warn("charge failure on unexpected subscription status — no status change applied",
			"subscription_id", subID,
			"status", currentStatus,
			"attempt", attemptNum)
	}
}

// applyPendingSchedule checks for a pending schedule and marks it as applied.
func (s *RecurrenceService) applyPendingSchedule(ctx context.Context, orgID, subID uuid.UUID) {
	schedule, err := s.schedules.GetPending(ctx, orgID, subID)
	if err != nil {
		// No pending schedule is the normal case.
		return
	}
	if err := s.schedules.Cancel(ctx, orgID, schedule.ID); err != nil {
		slog.Error("failed to apply pending schedule", "subscription_id", subID, "schedule_id", schedule.ID, "error", err)
	}
}
