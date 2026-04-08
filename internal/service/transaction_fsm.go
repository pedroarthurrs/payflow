package service

import (
	"context"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// TransactionFSMService orquestra a validação e persistência de transições de estado.
type TransactionFSMService struct {
	fsm   *domain.StateMachine
	txMgr domain.TxManager
	repo  domain.TransactionFSMRepository // repositório direto para leituras sem transação
}

// NewTransactionFSMService cria um TransactionFSMService.
// repo é usado para leituras diretas (ListHistory); txMgr é usado para escritas atômicas.
func NewTransactionFSMService(txMgr domain.TxManager, repo domain.TransactionFSMRepository) *TransactionFSMService {
	return &TransactionFSMService{
		fsm:   domain.NewStateMachine(),
		txMgr: txMgr,
		repo:  repo,
	}
}

// Transition aplica uma transição de estado para a transação especificada.
// Executa SELECT FOR UPDATE, valida via FSM e persiste status + histórico atomicamente.
func (s *TransactionFSMService) Transition(
	ctx context.Context,
	organizationID, transactionID uuid.UUID,
	toStatus domain.TransactionStatus,
	event domain.TransitionEvent,
) error {
	return s.txMgr.WithTx(ctx, func(tx domain.TxHandle) error {
		repo := tx.TransactionFSMRepo()

		txn, err := repo.GetForUpdate(ctx, organizationID, transactionID)
		if err != nil {
			return err
		}

		if err := s.fsm.Validate(txn.Status, toStatus); err != nil {
			return err
		}

		if err := repo.UpdateStatus(ctx, organizationID, transactionID, toStatus); err != nil {
			return err
		}

		return repo.InsertHistory(ctx, domain.TransitionHistory{
			OrganizationID: txn.OrganizationID,
			TransactionID:  transactionID,
			OldStatus:      &txn.Status,
			NewStatus:      toStatus,
			EventSource:    event.Source,
			EventSourceID:  event.SourceID,
			Reason:         event.Reason,
		})
	})
}

// AllowedTransitions delega para a FSM pura.
func (s *TransactionFSMService) AllowedTransitions(from domain.TransactionStatus) []domain.TransactionStatus {
	return s.fsm.AllowedTransitions(from)
}

// ListHistory retorna o histórico de transições de uma transação, ordenado por created_at ASC.
// Usa o repositório direto (sem transação SQL) pois é uma operação de leitura.
func (s *TransactionFSMService) ListHistory(
	ctx context.Context,
	organizationID, transactionID uuid.UUID,
) ([]domain.TransitionHistory, error) {
	return s.repo.ListHistory(ctx, organizationID, transactionID)
}
