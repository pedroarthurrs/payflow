package service

import (
	"context"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// PaymentService handles payment business logic, including CRUD operations
// and FSM-driven state transitions.
type PaymentService struct {
	payments domain.PaymentRepository
	fsm      *domain.StateMachine
	txMgr    domain.TxManager
	fsmRepo  domain.PaymentFSMRepository
}

// NewPaymentService creates a PaymentService.
func NewPaymentService(payments domain.PaymentRepository, txMgr domain.TxManager, fsmRepo domain.PaymentFSMRepository) *PaymentService {
	return &PaymentService{
		payments: payments,
		fsm:      domain.NewStateMachine(),
		txMgr:    txMgr,
		fsmRepo:  fsmRepo,
	}
}

// Create creates a new individual payment with status=pending for the given organization.
func (s *PaymentService) Create(ctx context.Context, organizationID uuid.UUID, input domain.PaymentInput) (*domain.Payment, error) {
	return s.payments.Create(ctx, organizationID, input)
}

// GetByID retrieves a single payment by ID for the given organization.
func (s *PaymentService) GetByID(ctx context.Context, organizationID, paymentID uuid.UUID) (*domain.Payment, error) {
	return s.payments.GetByID(ctx, organizationID, paymentID)
}

// List returns payments for the given organization with offset-based pagination and filters.
// Also returns the total count of matching payments for pagination metadata.
func (s *PaymentService) List(ctx context.Context, organizationID uuid.UUID, params domain.PaymentListParams) ([]domain.Payment, int64, error) {
	payments, err := s.payments.List(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	total, err := s.payments.Count(ctx, organizationID, params)
	if err != nil {
		return nil, 0, err
	}
	return payments, total, nil
}

// Transition applies a payment state transition atomically.
func (s *PaymentService) Transition(
	ctx context.Context,
	organizationID, paymentID uuid.UUID,
	toStatus domain.PaymentStatus,
	event domain.TransitionEvent,
) error {
	return s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		repo := tx.PaymentFSMRepo()

		payment, err := repo.GetForUpdate(ctx, organizationID, paymentID)
		if err != nil {
			return err
		}

		if err := s.fsm.Validate(payment.Status, toStatus); err != nil {
			return err
		}

		if err := repo.UpdateStatus(ctx, organizationID, paymentID, toStatus); err != nil {
			return err
		}

		return repo.InsertHistory(ctx, domain.TransitionHistory{
			OrganizationID: payment.OrganizationID,
			PaymentID:      paymentID,
			OldStatus:      &payment.Status,
			NewStatus:      toStatus,
			EventSource:    event.Source,
			EventSourceID:  event.SourceID,
			Reason:         event.Reason,
		})
	})
}

// AllowedTransitions returns valid next states from the given status.
func (s *PaymentService) AllowedTransitions(from domain.PaymentStatus) []domain.PaymentStatus {
	return s.fsm.AllowedTransitions(from)
}

// ListHistory returns the transition history of a payment ordered by created_at ASC.
func (s *PaymentService) ListHistory(
	ctx context.Context,
	organizationID, paymentID uuid.UUID,
) ([]domain.TransitionHistory, error) {
	return s.fsmRepo.ListHistory(ctx, organizationID, paymentID)
}
