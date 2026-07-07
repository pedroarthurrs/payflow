package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"

	paymentdb "payflow/internal/database/dbgen/payment"
	"payflow/internal/domain"

	"github.com/google/uuid"
	"github.com/sqlc-dev/pqtype"
)

// PaymentRepo implements domain.PaymentRepository backed by PostgreSQL via sqlc.
type PaymentRepo struct {
	q paymentdb.Querier
}

// NewPaymentRepo creates a PaymentRepo.
func NewPaymentRepo(q paymentdb.Querier) *PaymentRepo {
	return &PaymentRepo{q: q}
}

// Create inserts a new payment with status=pending and returns the created record.
func (r *PaymentRepo) Create(ctx context.Context, organizationID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
	currency := input.Currency
	if currency == "" {
		currency = "BRL"
	}

	row, err := r.q.CreatePayment(ctx, paymentdb.CreatePaymentParams{
		OrganizationID: organizationID,
		CustomerID:     input.CustomerID,
		SubscriptionID: toNullUUID(input.SubscriptionID),
		PaymentMethod:  sql.NullString{String: input.PaymentMethod, Valid: input.PaymentMethod != ""},
		AmountCents:    input.AmountCents,
		Status:         paymentdb.PaymentStatusTypePending,
		Currency:       sql.NullString{String: currency, Valid: true},
		IdempotencyKey: sql.NullString{String: input.IdempotencyKey, Valid: input.IdempotencyKey != ""},
	})
	if err != nil {
		slog.Error("failed to create payment", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}

	return toDomainPayment(row), nil
}

// GetByID retrieves a payment by ID, mapping sql.ErrNoRows to domain.ErrPaymentNotFound.
func (r *PaymentRepo) GetByID(ctx context.Context, organizationID, paymentID uuid.UUID) (*domain.Payment, error) {
	row, err := r.q.GetPaymentByID(ctx, paymentdb.GetPaymentByIDParams{
		OrganizationID: organizationID,
		ID:             paymentID,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, domain.ErrPaymentNotFound
		}
		slog.Error("failed to get payment by id", "organization_id", organizationID, "payment_id", paymentID, "error", err)
		return nil, domain.ErrInternal
	}

	return toDomainPayment(row), nil
}

// List returns payments for the given organization with offset-based pagination and filters.
func (r *PaymentRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.PaymentListParams) ([]domain.Payment, error) {
	arg := paymentdb.ListPaymentsParams{
		OrganizationID: organizationID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	}
	if params.Status != nil {
		arg.Status = paymentdb.NullPaymentStatusType{
			PaymentStatusType: paymentdb.PaymentStatusType(*params.Status),
			Valid:             true,
		}
	}
	if params.CustomerID != nil {
		arg.CustomerID = uuid.NullUUID{UUID: *params.CustomerID, Valid: true}
	}
	if params.PaymentMethod != nil {
		arg.PaymentMethod = sql.NullString{String: *params.PaymentMethod, Valid: true}
	}
	if params.CreatedFrom != nil {
		arg.CreatedFrom = sql.NullTime{Time: *params.CreatedFrom, Valid: true}
	}
	if params.CreatedTo != nil {
		arg.CreatedTo = sql.NullTime{Time: *params.CreatedTo, Valid: true}
	}

	rows, err := r.q.ListPayments(ctx, arg)
	if err != nil {
		slog.Error("failed to list payments", "organization_id", organizationID, "error", err)
		return nil, domain.ErrInternal
	}

	payments := make([]domain.Payment, len(rows))
	for i := range rows {
		payments[i] = *toDomainPayment(rows[i])
	}
	return payments, nil
}

func toDomainPayment(row paymentdb.Payment) *domain.Payment {
	p := &domain.Payment{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		CustomerID:     row.CustomerID,
		AmountCents:    row.AmountCents,
		Status:         domain.PaymentStatus(row.Status),
	}

	if row.InvoiceID.Valid {
		id := row.InvoiceID.UUID
		p.InvoiceID = &id
	}

	if row.SubscriptionID.Valid {
		id := row.SubscriptionID.UUID
		p.SubscriptionID = &id
	}

	if row.GatewayName.Valid {
		p.GatewayName = &row.GatewayName.String
	}

	if row.GatewayReference.Valid {
		p.GatewayReference = &row.GatewayReference.String
	}

	if row.PaymentMethod.Valid {
		p.PaymentMethod = &row.PaymentMethod.String
	}

	if row.Currency.Valid {
		p.Currency = &row.Currency.String
	}

	if row.IdempotencyKey.Valid {
		p.IdempotencyKey = &row.IdempotencyKey.String
	}

	if row.ErrorLog.Valid {
		p.ErrorLog = row.ErrorLog.RawMessage
	}

	if row.CreatedAt.Valid {
		p.CreatedAt = row.CreatedAt.Time
	}

	if row.UpdatedAt.Valid {
		p.UpdatedAt = row.UpdatedAt.Time
	}

	return p
}

// Count returns the total number of payments matching the given filters for the organization.
func (r *PaymentRepo) Count(ctx context.Context, organizationID uuid.UUID, params domain.PaymentListParams) (int64, error) {
	arg := paymentdb.CountPaymentsParams{
		OrganizationID: organizationID,
	}
	if params.Status != nil {
		arg.Status = paymentdb.NullPaymentStatusType{
			PaymentStatusType: paymentdb.PaymentStatusType(*params.Status),
			Valid:             true,
		}
	}
	if params.CustomerID != nil {
		arg.CustomerID = uuid.NullUUID{UUID: *params.CustomerID, Valid: true}
	}
	if params.PaymentMethod != nil {
		arg.PaymentMethod = sql.NullString{String: *params.PaymentMethod, Valid: true}
	}
	if params.CreatedFrom != nil {
		arg.CreatedFrom = sql.NullTime{Time: *params.CreatedFrom, Valid: true}
	}
	if params.CreatedTo != nil {
		arg.CreatedTo = sql.NullTime{Time: *params.CreatedTo, Valid: true}
	}

	count, err := r.q.CountPayments(ctx, arg)
	if err != nil {
		slog.Error("failed to count payments", "organization_id", organizationID, "error", err)
		return 0, domain.ErrInternal
	}
	return count, nil
}

// UpdateGatewayReference updates only the gateway_name and gateway_reference columns.
func (r *PaymentRepo) UpdateGatewayReference(ctx context.Context, organizationID, paymentID uuid.UUID, gatewayName, gatewayReference string) error {
	err := r.q.UpdatePaymentGatewayReference(ctx, paymentdb.UpdatePaymentGatewayReferenceParams{
		OrganizationID:   organizationID,
		ID:               paymentID,
		GatewayName:      sql.NullString{String: gatewayName, Valid: gatewayName != ""},
		GatewayReference: sql.NullString{String: gatewayReference, Valid: gatewayReference != ""},
	})
	if err != nil {
		slog.Error("failed to update gateway reference", "organization_id", organizationID, "payment_id", paymentID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

// UpdateErrorLog updates only the error_log column.
func (r *PaymentRepo) UpdateErrorLog(ctx context.Context, organizationID, paymentID uuid.UUID, errorLog json.RawMessage) error {
	err := r.q.UpdatePaymentErrorLog(ctx, paymentdb.UpdatePaymentErrorLogParams{
		OrganizationID: organizationID,
		ID:             paymentID,
		ErrorLog:       pqtype.NullRawMessage{RawMessage: errorLog, Valid: len(errorLog) > 0},
	})
	if err != nil {
		slog.Error("failed to update error log", "organization_id", organizationID, "payment_id", paymentID, "error", err)
		return domain.ErrInternal
	}
	return nil
}

var _ domain.PaymentRepository = (*PaymentRepo)(nil)

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
		CustomerID:     row.CustomerID,
		AmountCents:    row.AmountCents,
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

	if entry.OldStatus != nil {
		params.OldStatus = paymentdb.NullPaymentStatusType{
			PaymentStatusType: paymentdb.PaymentStatusType(*entry.OldStatus),
			Valid:             true,
		}
	}

	if entry.EventSource != "" {
		params.EventSource = sql.NullString{String: entry.EventSource, Valid: true}
	}

	if entry.EventSourceID != nil {
		params.EventSourceID = uuid.NullUUID{UUID: *entry.EventSourceID, Valid: true}
	}

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

func toDomainTransitionHistory(row paymentdb.ListPaymentHistoryRow) domain.TransitionHistory {
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
