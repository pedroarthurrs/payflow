package domain

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RecurringBillingJob is the payload stored in the queue for a recurring billing cycle.
type RecurringBillingJob struct {
	SubscriptionID uuid.UUID `json:"subscription_id"`
	OrganizationID uuid.UUID `json:"organization_id"`
}

// SubscriptionBillingCandidate is a minimal projection returned by ListDueForBilling.
type SubscriptionBillingCandidate struct {
	ID                uuid.UUID
	OrganizationID    uuid.UUID
	CustomerID        uuid.UUID
	Status            SubscriptionStatus
	CancelAtPeriodEnd bool
	CurrentPeriodEnd  time.Time
	PaymentMethod     string
}

// RecurrenceRepository defines cross-org operations used exclusively by the recurrence engine.
type RecurrenceRepository interface {
	// ListDueForBilling returns subscriptions whose billing period has ended and are not locked.
	// Uses SELECT FOR UPDATE SKIP LOCKED to prevent double processing across workers.
	ListDueForBilling(ctx context.Context, now time.Time, batchSize int) ([]SubscriptionBillingCandidate, error)
	// LockForBilling sets billing_locked_until to prevent the scheduler from re-enqueuing.
	LockForBilling(ctx context.Context, orgID, subID uuid.UUID, until time.Time) error
	// UnlockBilling clears billing_locked_until so the subscription can be retried.
	UnlockBilling(ctx context.Context, orgID, subID uuid.UUID) error
}
