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
		PaymentMethod:  sql.NullString{String: input.PaymentMethod, Valid: input.PaymentMethod != ""},
		AmountCents:    input.AmountCents,
		Currency:       sql.NullString{String: currency, Valid: true},
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

// List returns payments for the given organization with offset-based pagination.
func (r *PaymentRepo) List(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Payment, error) {
	rows, err := r.q.ListPayments(ctx, paymentdb.ListPaymentsParams{
		OrganizationID: organizationID,
		Limit:          params.Limit,
		Offset:         params.Offset,
	})
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
		AmountCents:    row.AmountCents,
		Status:         domain.PaymentStatus(row.Status),
	}

	if row.InvoiceID.Valid {
		id := row.InvoiceID.UUID
		p.InvoiceID = &id
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

var _ domain.PaymentRepository = (*PaymentRepo)(nil)
