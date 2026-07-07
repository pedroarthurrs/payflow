package service

import (
	"context"
	"log/slog"

	"payflow/internal/domain"

	"github.com/google/uuid"
)

// paymentLedger is the subset of LedgerService methods used by PaymentService.
// Using an interface allows test doubles to be injected without modifying callers.
type paymentLedger interface {
	CreditOnPaymentSuccess(ctx context.Context, tx domain.TxHandle, orgID uuid.UUID, p *domain.PaymentForUpdate) error
	LockOnPaymentDisputed(ctx context.Context, tx domain.TxHandle, orgID uuid.UUID, p *domain.PaymentForUpdate) error
	DebitOnPaymentRefund(ctx context.Context, tx domain.TxHandle, orgID uuid.UUID, p *domain.PaymentForUpdate) error
	ReleaseLockIfDisputed(ctx context.Context, tx domain.TxHandle, orgID uuid.UUID, p *domain.PaymentForUpdate) error
}

// PaymentService handles payment business logic, including CRUD operations
// and FSM-driven state transitions.
type PaymentService struct {
	payments domain.PaymentRepository
	fsm      *domain.StateMachine
	txMgr    domain.TxManager
	fsmRepo  domain.PaymentFSMRepository
	ledger   paymentLedger
}

// NewPaymentService creates a PaymentService.
func NewPaymentService(payments domain.PaymentRepository, txMgr domain.TxManager, fsmRepo domain.PaymentFSMRepository, ledger *LedgerService) *PaymentService {
	var l paymentLedger
	if ledger != nil {
		l = ledger
	}
	return &PaymentService{
		payments: payments,
		fsm:      domain.NewStateMachine(),
		txMgr:    txMgr,
		fsmRepo:  fsmRepo,
		ledger:   l,
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

		if err := repo.InsertHistory(ctx, domain.TransitionHistory{
			OrganizationID: payment.OrganizationID,
			PaymentID:      paymentID,
			OldStatus:      &payment.Status,
			NewStatus:      toStatus,
			EventSource:    event.Source,
			EventSourceID:  event.SourceID,
			Reason:         event.Reason,
		}); err != nil {
			return err
		}

		// Ledger hooks — non-fatal: log and continue on error
		if s.ledger != nil {
			switch toStatus { //nolint:exhaustive // pending/authorized/failed/canceled have no ledger action
			case domain.StatusSucceeded:
				if err := s.ledger.CreditOnPaymentSuccess(ctx, tx, organizationID, payment); err != nil {
					slog.Warn("ledger credit failed, payment still succeeded", "error", err, "payment_id", paymentID)
				}
				// If transitioning out of disputed, also release the lock
				if payment.Status == domain.StatusDisputed {
					if err := s.ledger.ReleaseLockIfDisputed(ctx, tx, organizationID, payment); err != nil {
						slog.Warn("failed to release dispute lock on succeeded", "error", err, "payment_id", paymentID)
					}
				}
			case domain.StatusDisputed:
				if err := s.ledger.LockOnPaymentDisputed(ctx, tx, organizationID, payment); err != nil {
					slog.Warn("ledger lock failed on disputed", "error", err, "payment_id", paymentID)
				}
			case domain.StatusRefunded, domain.StatusPartiallyRefunded:
				if err := s.ledger.DebitOnPaymentRefund(ctx, tx, organizationID, payment); err != nil {
					slog.Warn("ledger debit failed on refund", "error", err, "payment_id", paymentID)
				}
			default:
				// pending, authorized, failed, canceled — no ledger action
			}
		}

		return nil
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
