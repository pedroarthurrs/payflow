package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SubscriptionStatus represents the lifecycle state of a subscription.
type SubscriptionStatus string

// Subscription status constants.
const (
	SubStatusTrialing   SubscriptionStatus = "trialing"
	SubStatusActive     SubscriptionStatus = "active"
	SubStatusPastDue    SubscriptionStatus = "past_due"
	SubStatusUnpaid     SubscriptionStatus = "unpaid"
	SubStatusCanceled   SubscriptionStatus = "canceled"
	SubStatusPaused     SubscriptionStatus = "paused"
	SubStatusIncomplete SubscriptionStatus = "incomplete"
)

// SubscriptionRepository defines persistence for subscriptions.
type SubscriptionRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateSubscriptionInput) (*Subscription, error)
	GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*Subscription, error)
	List(ctx context.Context, organizationID uuid.UUID, params SubscriptionListParams) ([]Subscription, error)
	Count(ctx context.Context, organizationID uuid.UUID, params SubscriptionListParams) (int64, error)
	UpdateStatus(ctx context.Context, organizationID, subscriptionID uuid.UUID, newStatus SubscriptionStatus, reason string) (*Subscription, error)
	SetCancelAtPeriodEnd(ctx context.Context, organizationID, subscriptionID uuid.UUID, value bool) (*Subscription, error)
	// AdvancePeriod moves the subscription to the next billing period and clears billing_locked_until.
	AdvancePeriod(ctx context.Context, orgID, subID uuid.UUID, newStart, newEnd time.Time) (*Subscription, error)
}

// SubscriptionItemRepository defines persistence for subscription items.
type SubscriptionItemRepository interface {
	CreateItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []SubscriptionItemInput) ([]SubscriptionItem, error)
	ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []SubscriptionItemInput) ([]SubscriptionItem, error)
	ListBySubscription(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]SubscriptionItem, error)
}

// SubscriptionScheduleRepository defines persistence for subscription schedules.
type SubscriptionScheduleRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input CreateScheduleInput) (*SubscriptionSchedule, error)
	GetPending(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*SubscriptionSchedule, error)
	Cancel(ctx context.Context, organizationID, scheduleID uuid.UUID) error
}

// SubscriptionHistoryRepository defines read access to subscription status history.
type SubscriptionHistoryRepository interface {
	List(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]SubscriptionStatusHistory, error)
}

// Subscription represents a recurring billing agreement.
type Subscription struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	CustomerID         uuid.UUID
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
	CancelAtPeriodEnd  bool
	PaymentMethod      string
	BillingLockedUntil *time.Time
	Metadata           json.RawMessage
	Items              []SubscriptionItem
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// SubscriptionItem links a plan to a subscription.
type SubscriptionItem struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	PlanID         uuid.UUID
	Quantity       int32
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// SubscriptionStatusHistory records every status transition.
type SubscriptionStatusHistory struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	OldStatus      *SubscriptionStatus
	NewStatus      SubscriptionStatus
	Reason         string
	CreatedAt      time.Time
}

// SubscriptionSchedule represents a future change to a subscription.
type SubscriptionSchedule struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	SubscriptionID uuid.UUID
	ScheduledAt    time.Time
	TargetStatus   *SubscriptionStatus
	TargetPlanID   *uuid.UUID
	AppliedAt      *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// CreateSubscriptionInput carries the data required to create a subscription.
// Status, CurrentPeriodStart and CurrentPeriodEnd are computed by the service
// before calling the repository.
type CreateSubscriptionInput struct {
	CustomerID    uuid.UUID
	TrialEnd      *time.Time
	Metadata      json.RawMessage
	Items         []SubscriptionItemInput
	PaymentMethod string
	// Computed by service
	Status             SubscriptionStatus
	CurrentPeriodStart time.Time
	CurrentPeriodEnd   time.Time
}

// SubscriptionItemInput carries the data for a single subscription item.
type SubscriptionItemInput struct {
	PlanID   uuid.UUID
	Quantity int32
}

// CreateScheduleInput carries the data required to create a schedule.
type CreateScheduleInput struct {
	SubscriptionID uuid.UUID
	ScheduledAt    time.Time
	TargetStatus   *SubscriptionStatus
	TargetPlanID   *uuid.UUID
}

// SubscriptionListParams carries filter and pagination parameters for listing subscriptions.
type SubscriptionListParams struct {
	ListParams
	CustomerID  *uuid.UUID
	Status      []SubscriptionStatus
	CreatedFrom *time.Time
	CreatedTo   *time.Time
}

// SubscriptionStateMachine is the pure FSM for subscriptions — no infrastructure dependencies.
type SubscriptionStateMachine struct{}

// subscriptionTransitionRules is the immutable graph of valid transitions.
var subscriptionTransitionRules = map[SubscriptionStatus][]SubscriptionStatus{
	SubStatusTrialing:   {SubStatusActive, SubStatusCanceled},
	SubStatusActive:     {SubStatusPastDue, SubStatusCanceled, SubStatusPaused},
	SubStatusPastDue:    {SubStatusActive, SubStatusUnpaid, SubStatusCanceled},
	SubStatusUnpaid:     {SubStatusActive, SubStatusCanceled},
	SubStatusPaused:     {SubStatusActive, SubStatusCanceled},
	SubStatusIncomplete: {SubStatusActive, SubStatusCanceled},
	SubStatusCanceled:   {},
}

// subscriptionTerminalStates are irreversible statuses.
var subscriptionTerminalStates = map[SubscriptionStatus]bool{
	SubStatusCanceled: true,
}

// NewSubscriptionStateMachine creates a SubscriptionStateMachine.
func NewSubscriptionStateMachine() *SubscriptionStateMachine {
	return &SubscriptionStateMachine{}
}

// Validate returns nil if the transition from → to is valid, or an error otherwise.
func (sm *SubscriptionStateMachine) Validate(from, to SubscriptionStatus) error {
	if from == to {
		return ErrSameStateTransition
	}
	if subscriptionTerminalStates[from] {
		return ErrSubscriptionCanceled
	}
	for _, allowed := range subscriptionTransitionRules[from] {
		if allowed == to {
			return nil
		}
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidSubscriptionTransition, from, to)
}

// IsTerminal returns true if status is a terminal state.
func (sm *SubscriptionStateMachine) IsTerminal(status SubscriptionStatus) bool {
	return subscriptionTerminalStates[status]
}
