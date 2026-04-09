package service

import (
	"context"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// PaymentFSMService orquestra a validação e persistência de transições de estado.
type PaymentFSMService struct {
	fsm   *domain.StateMachine
	txMgr domain.TxManager
	repo  domain.PaymentFSMRepository // repositório direto para leituras sem transação
}

// NewPaymentFSMService cria um PaymentFSMService.
// repo é usado para leituras diretas (ListHistory); txMgr é usado para escritas atômicas.
func NewPaymentFSMService(txMgr domain.TxManager, repo domain.PaymentFSMRepository) *PaymentFSMService {
	return &PaymentFSMService{
		fsm:   domain.NewStateMachine(),
		txMgr: txMgr,
		repo:  repo,
	}
}

// Transition aplica uma transição de estado para o pagamento especificado.
// Executa SELECT FOR UPDATE, valida via FSM e persiste status + histórico atomicamente.
func (s *PaymentFSMService) Transition(
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

// AllowedTransitions delega para a FSM pura.
func (s *PaymentFSMService) AllowedTransitions(from domain.PaymentStatus) []domain.PaymentStatus {
	return s.fsm.AllowedTransitions(from)
}

// ListHistory retorna o histórico de transições de um pagamento, ordenado por created_at ASC.
// Usa o repositório direto (sem transação SQL) pois é uma operação de leitura.
func (s *PaymentFSMService) ListHistory(
	ctx context.Context,
	organizationID, paymentID uuid.UUID,
) ([]domain.TransitionHistory, error) {
	return s.repo.ListHistory(ctx, organizationID, paymentID)
}
