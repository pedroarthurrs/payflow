package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	paymentdb "i9pay/internal/database/dbgen/payment"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// PaymentFSMRepo implements domain.PaymentFSMRepository backed by PostgreSQL via sqlc.
type PaymentFSMRepo struct {
	q paymentdb.Querier
}

// NewPaymentFSMRepo creates a PaymentFSMRepo.
func NewPaymentFSMRepo(q paymentdb.Querier) *PaymentFSMRepo {
	return &PaymentFSMRepo{q: q}
}

// GetForUpdate retrieves a payment with SELECT FOR UPDATE, mapping sql.ErrNoRows to domain.ErrPaymentNotFound.
func (r *PaymentFSMRepo) GetForUpdate(ctx context.Context, orgID, paymentID uuid.UUID) (*domain.PaymentForUpdate, error) {
	row, err := r.q.GetPaymentForUpdate(ctx, paymentdb.GetPaymentForUpdateParams{
		OrganizationID: orgID,
		ID:             paymentID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		slog.Error("failed to get payment for update", "organization_id", orgID, "payment_id", paymentID, "error", err)
		return nil, domain.ErrInternal
	}
	return &domain.PaymentForUpdate{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Status:         domain.PaymentStatus(row.Status),
	}, nil
}

// UpdateStatus updates the payment status.
func (r *PaymentFSMRepo) UpdateStatus(ctx context.Context, orgID, paymentID uuid.UUID, newStatus domain.PaymentStatus) error {
	err := r.q.UpdatePaymentStatus(ctx, paymentdb.UpdatePaymentStatusParams{
		OrganizationID: orgID,
		ID:             paymentID,
		Status:         paymentdb.PaymentStatusType(newStatus),
	})
	if err != nil {
		slog.Error("failed to update payment status", "organization_id", orgID, "payment_id", paymentID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// InsertHistory persists a transition history record.
func (r *PaymentFSMRepo) InsertHistory(ctx context.Context, entry domain.TransitionHistory) error {
	params := paymentdb.InsertPaymentHistoryParams{
		OrganizationID: entry.OrganizationID,
		PaymentID:      entry.PaymentID,
		NewStatus:      paymentdb.PaymentStatusType(entry.NewStatus),
	}

	// Map *domain.PaymentStatus → NullPaymentStatusType
	if entry.OldStatus != nil {
		params.OldStatus = paymentdb.NullPaymentStatusType{
			PaymentStatusType: paymentdb.PaymentStatusType(*entry.OldStatus),
			Valid:             true,
		}
	}

	// Map optional EventSource
	if entry.EventSource != "" {
		params.EventSource = sql.NullString{String: entry.EventSource, Valid: true}
	}

	// Map optional EventSourceID
	if entry.EventSourceID != nil {
		params.EventSourceID = uuid.NullUUID{UUID: *entry.EventSourceID, Valid: true}
	}

	// Map optional Reason
	if entry.Reason != "" {
		params.Reason = sql.NullString{String: entry.Reason, Valid: true}
	}

	if err := r.q.InsertPaymentHistory(ctx, params); err != nil {
		slog.Error("failed to insert payment history", "organization_id", entry.OrganizationID, "payment_id", entry.PaymentID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// ListHistory returns the transition history for a payment, ordered by created_at ASC.
func (r *PaymentFSMRepo) ListHistory(ctx context.Context, orgID, paymentID uuid.UUID) ([]domain.TransitionHistory, error) {
	rows, err := r.q.ListPaymentHistory(ctx, paymentdb.ListPaymentHistoryParams{
		OrganizationID: orgID,
		PaymentID:      paymentID,
	})
	if err != nil {
		slog.Error("failed to list payment history", "organization_id", orgID, "payment_id", paymentID, "error", err)
		return nil, domain.ErrInternal
	}

	records := make([]domain.TransitionHistory, len(rows))
	for i := range rows {
		records[i] = toDomainTransitionHistory(rows[i])
	}
	return records, nil
}

func toDomainTransitionHistory(row paymentdb.PaymentStatusHistory) domain.TransitionHistory {
	h := domain.TransitionHistory{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		PaymentID:      row.PaymentID,
		NewStatus:      domain.PaymentStatus(row.NewStatus),
	}

	if row.OldStatus.Valid {
		s := domain.PaymentStatus(row.OldStatus.PaymentStatusType)
		h.OldStatus = &s
	}

	if row.EventSource.Valid {
		h.EventSource = row.EventSource.String
	}

	if row.EventSourceID.Valid {
		id := row.EventSourceID.UUID
		h.EventSourceID = &id
	}

	if row.Reason.Valid {
		h.Reason = row.Reason.String
	}

	if row.CreatedAt.Valid {
		h.CreatedAt = row.CreatedAt.Time
	}

	return h
}

var _ domain.PaymentFSMRepository = (*PaymentFSMRepo)(nil)
