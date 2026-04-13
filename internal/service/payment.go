package service

import (
	"context"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// PaymentService handles individual payment business logic.
type PaymentService struct {
	payments domain.PaymentRepository
}

// NewPaymentService creates a PaymentService.
func NewPaymentService(payments domain.PaymentRepository) *PaymentService {
	return &PaymentService{payments: payments}
}

// Create creates a new individual payment with status=pending for the given organization.
func (s *PaymentService) Create(ctx context.Context, organizationID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
	return s.payments.Create(ctx, organizationID, input)
}

// GetByID retrieves a single payment by ID for the given organization.
func (s *PaymentService) GetByID(ctx context.Context, organizationID, paymentID uuid.UUID) (*domain.Payment, error) {
	return s.payments.GetByID(ctx, organizationID, paymentID)
}

// List returns payments for the given organization with offset-based pagination.
func (s *PaymentService) List(ctx context.Context, organizationID uuid.UUID, params domain.ListParams) ([]domain.Payment, error) {
	return s.payments.List(ctx, organizationID, params)
}
