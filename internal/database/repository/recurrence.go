package repository

import (
	"context"
	"database/sql"
	"log/slog"
	"time"

	recurrencedb "i9pay/internal/database/dbgen/recurrence"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// RecurrenceRepo implements domain.RecurrenceRepository.
type RecurrenceRepo struct {
	db *sql.DB
	q  recurrencedb.Querier
}

// NewRecurrenceRepo creates a RecurrenceRepo.
func NewRecurrenceRepo(db *sql.DB, q recurrencedb.Querier) *RecurrenceRepo {
	return &RecurrenceRepo{db: db, q: q}
}

// ListDueForBilling scans for subscriptions whose billing period has ended, locks them
// for processing using SELECT FOR UPDATE SKIP LOCKED + UPDATE billing_locked_until,
// and returns the candidates.
func (r *RecurrenceRepo) ListDueForBilling(ctx context.Context, now time.Time, batchSize int) ([]domain.SubscriptionBillingCandidate, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		slog.Error("failed to begin recurrence scan transaction", "error", err)
		return nil, domain.ErrInternal
	}
	defer func() { _ = tx.Rollback() }()

	qTx := recurrencedb.New(tx)

	rows, err := qTx.ListSubscriptionsDueForBilling(ctx, recurrencedb.ListSubscriptionsDueForBillingParams{
		CurrentPeriodEnd: now,
		Limit:            int32(batchSize),
	})
	if err != nil {
		slog.Error("failed to list subscriptions due for billing", "error", err)
		return nil, domain.ErrInternal
	}

	lockUntil := now.Add(30 * time.Minute)
	candidates := make([]domain.SubscriptionBillingCandidate, 0, len(rows))
	for _, row := range rows {
		if err := qTx.LockSubscriptionForBilling(ctx, recurrencedb.LockSubscriptionForBillingParams{
			OrganizationID:     row.OrganizationID,
			ID:                 row.ID,
			BillingLockedUntil: sql.NullTime{Time: lockUntil, Valid: true},
		}); err != nil {
			slog.Error("failed to lock subscription for billing", "subscription_id", row.ID, "error", err)
			return nil, domain.ErrInternal
		}
		candidates = append(candidates, domain.SubscriptionBillingCandidate{
			ID:                row.ID,
			OrganizationID:    row.OrganizationID,
			CustomerID:        row.CustomerID,
			Status:            domain.SubscriptionStatus(row.Status),
			CancelAtPeriodEnd: row.CancelAtPeriodEnd.Bool,
			CurrentPeriodEnd:  row.CurrentPeriodEnd,
			PaymentMethod:     row.PaymentMethod,
		})
	}

	if err := tx.Commit(); err != nil {
		slog.Error("failed to commit recurrence scan transaction", "error", err)
		return nil, domain.ErrInternal
	}
	return candidates, nil
}

// LockForBilling sets billing_locked_until to prevent re-enqueueing.
func (r *RecurrenceRepo) LockForBilling(ctx context.Context, orgID, subID uuid.UUID, until time.Time) error {
	if err := r.q.LockSubscriptionForBilling(ctx, recurrencedb.LockSubscriptionForBillingParams{
		OrganizationID:     orgID,
		ID:                 subID,
		BillingLockedUntil: sql.NullTime{Time: until, Valid: true},
	}); err != nil {
		slog.Error("failed to lock subscription for billing", "organization_id", orgID, "subscription_id", subID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// UnlockBilling clears billing_locked_until so the subscription can be retried.
func (r *RecurrenceRepo) UnlockBilling(ctx context.Context, orgID, subID uuid.UUID) error {
	if err := r.q.UnlockSubscriptionBilling(ctx, recurrencedb.UnlockSubscriptionBillingParams{
		OrganizationID: orgID,
		ID:             subID,
	}); err != nil {
		slog.Error("failed to unlock subscription billing", "organization_id", orgID, "subscription_id", subID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

var _ domain.RecurrenceRepository = (*RecurrenceRepo)(nil)
