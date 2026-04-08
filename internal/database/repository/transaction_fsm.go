package repository

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	transactiondb "i9pay/internal/database/dbgen/transaction"
	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// TransactionFSMRepo implements domain.TransactionFSMRepository backed by PostgreSQL via sqlc.
type TransactionFSMRepo struct {
	q transactiondb.Querier
}

// NewTransactionFSMRepo creates a TransactionFSMRepo.
func NewTransactionFSMRepo(q transactiondb.Querier) *TransactionFSMRepo {
	return &TransactionFSMRepo{q: q}
}

// GetForUpdate retrieves a transaction with SELECT FOR UPDATE, mapping sql.ErrNoRows to domain.ErrTransactionNotFound.
func (r *TransactionFSMRepo) GetForUpdate(ctx context.Context, orgID, txID uuid.UUID) (*domain.TransactionForUpdate, error) {
	row, err := r.q.GetTransactionForUpdate(ctx, transactiondb.GetTransactionForUpdateParams{
		OrganizationID: orgID,
		ID:             txID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrTransactionNotFound
		}
		slog.Error("failed to get transaction for update", "organization_id", orgID, "transaction_id", txID, "error", err)
		return nil, domain.ErrInternal
	}
	return &domain.TransactionForUpdate{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Status:         domain.TransactionStatus(row.Status),
	}, nil
}

// UpdateStatus updates the transaction status.
func (r *TransactionFSMRepo) UpdateStatus(ctx context.Context, orgID, txID uuid.UUID, newStatus domain.TransactionStatus) error {
	err := r.q.UpdateTransactionStatus(ctx, transactiondb.UpdateTransactionStatusParams{
		OrganizationID: orgID,
		ID:             txID,
		Status:         transactiondb.TransactionStatusType(newStatus),
	})
	if err != nil {
		slog.Error("failed to update transaction status", "organization_id", orgID, "transaction_id", txID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// InsertHistory persists a transition history record.
func (r *TransactionFSMRepo) InsertHistory(ctx context.Context, entry domain.TransitionHistory) error {
	params := transactiondb.InsertTransitionHistoryParams{
		OrganizationID: entry.OrganizationID,
		TransactionID:  entry.TransactionID,
		NewStatus:      transactiondb.TransactionStatusType(entry.NewStatus),
	}

	// Map *domain.TransactionStatus → NullTransactionStatusType
	if entry.OldStatus != nil {
		params.OldStatus = transactiondb.NullTransactionStatusType{
			TransactionStatusType: transactiondb.TransactionStatusType(*entry.OldStatus),
			Valid:                 true,
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

	if err := r.q.InsertTransitionHistory(ctx, params); err != nil {
		slog.Error("failed to insert transition history", "organization_id", entry.OrganizationID, "transaction_id", entry.TransactionID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// ListHistory returns the transition history for a transaction, ordered by created_at ASC.
func (r *TransactionFSMRepo) ListHistory(ctx context.Context, orgID, txID uuid.UUID) ([]domain.TransitionHistory, error) {
	rows, err := r.q.ListTransitionHistory(ctx, transactiondb.ListTransitionHistoryParams{
		OrganizationID: orgID,
		TransactionID:  txID,
	})
	if err != nil {
		slog.Error("failed to list transition history", "organization_id", orgID, "transaction_id", txID, "error", err)
		return nil, domain.ErrInternal
	}

	records := make([]domain.TransitionHistory, len(rows))
	for i := range rows {
		records[i] = toDomainTransitionHistory(rows[i])
	}
	return records, nil
}

func toDomainTransitionHistory(row transactiondb.TransactionStatusHistory) domain.TransitionHistory {
	h := domain.TransitionHistory{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		TransactionID:  row.TransactionID,
		NewStatus:      domain.TransactionStatus(row.NewStatus),
	}

	if row.OldStatus.Valid {
		s := domain.TransactionStatus(row.OldStatus.TransactionStatusType)
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

var _ domain.TransactionFSMRepository = (*TransactionFSMRepo)(nil)
