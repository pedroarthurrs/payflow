package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	subscriptiondb "i9pay/internal/database/dbgen/subscription"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// =============================================================================
// SubscriptionRepo
// =============================================================================

// SubscriptionRepo implements domain.SubscriptionRepository.
type SubscriptionRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionRepo creates a SubscriptionRepo.
func NewSubscriptionRepo(q subscriptiondb.Querier) *SubscriptionRepo {
	return &SubscriptionRepo{q: q}
}

// Create inserts a new subscription (without items — items are inserted separately inside a transaction).
func (r *SubscriptionRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateSubscriptionInput) (*domain.Subscription, error) {
	row, err := r.q.CreateSubscription(ctx, subscriptiondb.CreateSubscriptionParams{
		OrganizationID:     organizationID,
		CustomerID:         input.CustomerID,
		Status:             subscriptiondb.SubscriptionStatusType(input.Status),
		CurrentPeriodStart: input.CurrentPeriodStart,
		CurrentPeriodEnd:   input.CurrentPeriodEnd,
		Metadata:           toNullRawMessage(input.Metadata),
	})
	if err != nil {
		slog.Error("failed to create subscription", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// GetByID retrieves a subscription by ID.
func (r *SubscriptionRepo) GetByID(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.Subscription, error) {
	row, err := r.q.GetSubscriptionByID(ctx, subscriptiondb.GetSubscriptionByIDParams{
		OrganizationID: organizationID,
		ID:             subscriptionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		slog.Error("failed to get subscription", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// List returns subscriptions for an organization with optional filters.
func (r *SubscriptionRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.SubscriptionListParams) ([]domain.Subscription, error) {
	rows, err := r.q.ListSubscriptions(ctx, subscriptiondb.ListSubscriptionsParams{
		OrganizationID: organizationID,
		CustomerID:     toNullUUID(params.CustomerID),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
	if err != nil {
		slog.Error("failed to list subscriptions", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}

	subs := make([]domain.Subscription, 0, len(rows))
	for i := range rows {
		s := toDomainSubscription(rows[i])
		// Apply in-memory status filter when provided
		if len(params.Status) > 0 && !containsStatus(params.Status, s.Status) {
			continue
		}
		subs = append(subs, s)
	}
	return subs, nil
}

// Count returns the total count matching the filters (without status filtering — done in-memory).
func (r *SubscriptionRepo) Count(ctx context.Context, organizationID uuid.UUID, params domain.SubscriptionListParams) (int64, error) {
	count, err := r.q.CountSubscriptions(ctx, subscriptiondb.CountSubscriptionsParams{
		OrganizationID: organizationID,
		CustomerID:     toNullUUID(params.CustomerID),
		CreatedFrom:    toNullTime(params.CreatedFrom),
		CreatedTo:      toNullTime(params.CreatedTo),
	})
	if err != nil {
		slog.Error("failed to count subscriptions", "organization_id", organizationID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// UpdateStatus transitions the subscription to a new status.
func (r *SubscriptionRepo) UpdateStatus(ctx context.Context, organizationID, subscriptionID uuid.UUID, newStatus domain.SubscriptionStatus, _ string) (*domain.Subscription, error) {
	row, err := r.q.UpdateSubscriptionStatus(ctx, subscriptiondb.UpdateSubscriptionStatusParams{
		OrganizationID: organizationID,
		ID:             subscriptionID,
		Status:         subscriptiondb.SubscriptionStatusType(newStatus),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		slog.Error("failed to update subscription status", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// SetCancelAtPeriodEnd updates the cancel_at_period_end flag.
func (r *SubscriptionRepo) SetCancelAtPeriodEnd(ctx context.Context, organizationID, subscriptionID uuid.UUID, value bool) (*domain.Subscription, error) {
	row, err := r.q.SetCancelAtPeriodEnd(ctx, subscriptiondb.SetCancelAtPeriodEndParams{
		OrganizationID:    organizationID,
		ID:                subscriptionID,
		CancelAtPeriodEnd: sql.NullBool{Bool: value, Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrSubscriptionNotFound
		}
		slog.Error("failed to set cancel_at_period_end", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSubscription(row)
	return &s, nil
}

// =============================================================================
// SubscriptionItemRepo
// =============================================================================

// SubscriptionItemRepo implements domain.SubscriptionItemRepository.
type SubscriptionItemRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionItemRepo creates a SubscriptionItemRepo.
func NewSubscriptionItemRepo(q subscriptiondb.Querier) *SubscriptionItemRepo {
	return &SubscriptionItemRepo{q: q}
}

// CreateItems inserts multiple items for a subscription.
func (r *SubscriptionItemRepo) CreateItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	result := make([]domain.SubscriptionItem, 0, len(items))
	for _, item := range items {
		row, err := r.q.CreateSubscriptionItem(ctx, subscriptiondb.CreateSubscriptionItemParams{
			OrganizationID: organizationID,
			SubscriptionID: subscriptionID,
			PlanID:         item.PlanID,
			Quantity:       sql.NullInt32{Int32: item.Quantity, Valid: true},
		})
		if err != nil {
			slog.Error("failed to create subscription item", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
			return nil, domain.ErrInternal
		}
		result = append(result, toDomainSubscriptionItem(row))
	}
	return result, nil
}

// ReplaceItems deletes existing items and inserts the new set.
// Must be called inside a transaction (txHandle.SubscriptionItemRepo()).
func (r *SubscriptionItemRepo) ReplaceItems(ctx context.Context, organizationID, subscriptionID uuid.UUID, items []domain.SubscriptionItemInput) ([]domain.SubscriptionItem, error) {
	if err := r.q.DeleteSubscriptionItems(ctx, subscriptiondb.DeleteSubscriptionItemsParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	}); err != nil {
		slog.Error("failed to delete subscription items", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	return r.CreateItems(ctx, organizationID, subscriptionID, items)
}

// ListBySubscription returns all items for a subscription.
func (r *SubscriptionItemRepo) ListBySubscription(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionItem, error) {
	rows, err := r.q.ListSubscriptionItems(ctx, subscriptiondb.ListSubscriptionItemsParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		slog.Error("failed to list subscription items", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	result := make([]domain.SubscriptionItem, len(rows))
	for i := range rows {
		result[i] = toDomainSubscriptionItem(rows[i])
	}
	return result, nil
}

// =============================================================================
// SubscriptionScheduleRepo
// =============================================================================

// SubscriptionScheduleRepo implements domain.SubscriptionScheduleRepository.
type SubscriptionScheduleRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionScheduleRepo creates a SubscriptionScheduleRepo.
func NewSubscriptionScheduleRepo(q subscriptiondb.Querier) *SubscriptionScheduleRepo {
	return &SubscriptionScheduleRepo{q: q}
}

// Create inserts a new schedule.
func (r *SubscriptionScheduleRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.CreateScheduleInput) (*domain.SubscriptionSchedule, error) {
	params := subscriptiondb.CreateSubscriptionScheduleParams{
		OrganizationID: organizationID,
		SubscriptionID: input.SubscriptionID,
		ScheduledAt:    input.ScheduledAt,
	}
	if input.TargetStatus != nil {
		params.TargetStatus = subscriptiondb.NullSubscriptionStatusType{
			SubscriptionStatusType: subscriptiondb.SubscriptionStatusType(*input.TargetStatus),
			Valid:                  true,
		}
	}
	if input.TargetPlanID != nil {
		params.TargetPlanID = uuid.NullUUID{UUID: *input.TargetPlanID, Valid: true}
	}
	row, err := r.q.CreateSubscriptionSchedule(ctx, params)
	if err != nil {
		slog.Error("failed to create subscription schedule", "organization_id", organizationID, "subscription_id", input.SubscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSchedule(row)
	return &s, nil
}

// GetPending retrieves the pending (unapplied) schedule for a subscription.
func (r *SubscriptionScheduleRepo) GetPending(ctx context.Context, organizationID, subscriptionID uuid.UUID) (*domain.SubscriptionSchedule, error) {
	row, err := r.q.GetPendingSchedule(ctx, subscriptiondb.GetPendingScheduleParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrScheduleNotFound
		}
		slog.Error("failed to get pending schedule", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	s := toDomainSchedule(row)
	return &s, nil
}

// Cancel marks a schedule as applied (effectively canceling it without deleting).
func (r *SubscriptionScheduleRepo) Cancel(ctx context.Context, organizationID, scheduleID uuid.UUID) error {
	if err := r.q.CancelSchedule(ctx, subscriptiondb.CancelScheduleParams{
		OrganizationID: organizationID,
		ID:             scheduleID,
	}); err != nil {
		slog.Error("failed to cancel schedule", "organization_id", organizationID, "schedule_id", scheduleID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// =============================================================================
// SubscriptionHistoryRepo
// =============================================================================

// SubscriptionHistoryRepo implements domain.SubscriptionHistoryRepository.
type SubscriptionHistoryRepo struct {
	q subscriptiondb.Querier
}

// NewSubscriptionHistoryRepo creates a SubscriptionHistoryRepo.
func NewSubscriptionHistoryRepo(q subscriptiondb.Querier) *SubscriptionHistoryRepo {
	return &SubscriptionHistoryRepo{q: q}
}

// List returns all history entries for a subscription, ordered by created_at ASC.
func (r *SubscriptionHistoryRepo) List(ctx context.Context, organizationID, subscriptionID uuid.UUID) ([]domain.SubscriptionStatusHistory, error) {
	rows, err := r.q.ListSubscriptionHistory(ctx, subscriptiondb.ListSubscriptionHistoryParams{
		OrganizationID: organizationID,
		SubscriptionID: subscriptionID,
	})
	if err != nil {
		slog.Error("failed to list subscription history", "organization_id", organizationID, "subscription_id", subscriptionID, "error", err)
		return nil, domain.ErrInternal
	}
	result := make([]domain.SubscriptionStatusHistory, len(rows))
	for i := range rows {
		result[i] = toDomainHistory(rows[i])
	}
	return result, nil
}

// =============================================================================
// Mapping helpers
// =============================================================================

func toDomainSubscription(row subscriptiondb.Subscription) domain.Subscription {
	s := domain.Subscription{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		CustomerID:         row.CustomerID,
		Status:             domain.SubscriptionStatus(row.Status),
		CurrentPeriodStart: row.CurrentPeriodStart,
		CurrentPeriodEnd:   row.CurrentPeriodEnd,
	}
	if row.CancelAtPeriodEnd.Valid {
		s.CancelAtPeriodEnd = row.CancelAtPeriodEnd.Bool
	}
	if row.Metadata.Valid {
		s.Metadata = row.Metadata.RawMessage
	}
	if row.CreatedAt.Valid {
		s.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		s.UpdatedAt = row.UpdatedAt.Time
	}
	return s
}

func toDomainSubscriptionItem(row subscriptiondb.SubscriptionItem) domain.SubscriptionItem {
	item := domain.SubscriptionItem{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		PlanID:         row.PlanID,
		Quantity:       1,
	}
	if row.Quantity.Valid {
		item.Quantity = row.Quantity.Int32
	}
	if row.CreatedAt.Valid {
		item.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		item.UpdatedAt = row.UpdatedAt.Time
	}
	return item
}

func toDomainHistory(row subscriptiondb.SubscriptionStatusHistory) domain.SubscriptionStatusHistory {
	h := domain.SubscriptionStatusHistory{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		NewStatus:      domain.SubscriptionStatus(row.NewStatus),
	}
	if row.OldStatus.Valid {
		s := domain.SubscriptionStatus(row.OldStatus.SubscriptionStatusType)
		h.OldStatus = &s
	}
	if row.Reason.Valid {
		h.Reason = row.Reason.String
	}
	if row.CreatedAt.Valid {
		h.CreatedAt = row.CreatedAt.Time
	}
	return h
}

func toDomainSchedule(row subscriptiondb.SubscriptionSchedule) domain.SubscriptionSchedule {
	s := domain.SubscriptionSchedule{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		SubscriptionID: row.SubscriptionID,
		ScheduledAt:    row.ScheduledAt,
	}
	if row.TargetStatus.Valid {
		st := domain.SubscriptionStatus(row.TargetStatus.SubscriptionStatusType)
		s.TargetStatus = &st
	}
	if row.TargetPlanID.Valid {
		s.TargetPlanID = &row.TargetPlanID.UUID
	}
	if row.AppliedAt.Valid {
		s.AppliedAt = &row.AppliedAt.Time
	}
	if row.CreatedAt.Valid {
		s.CreatedAt = row.CreatedAt.Time
	}
	if row.UpdatedAt.Valid {
		s.UpdatedAt = row.UpdatedAt.Time
	}
	return s
}

func containsStatus(statuses []domain.SubscriptionStatus, s domain.SubscriptionStatus) bool {
	for _, v := range statuses {
		if v == s {
			return true
		}
	}
	return false
}

// Ensure interfaces are satisfied at compile time.
var _ domain.SubscriptionRepository = (*SubscriptionRepo)(nil)
var _ domain.SubscriptionItemRepository = (*SubscriptionItemRepo)(nil)
var _ domain.SubscriptionScheduleRepository = (*SubscriptionScheduleRepo)(nil)
var _ domain.SubscriptionHistoryRepository = (*SubscriptionHistoryRepo)(nil)
