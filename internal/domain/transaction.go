package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// TransactionStatus representa os estados possíveis de uma transação.
type TransactionStatus string

// Transaction status constants representing all possible states of a transaction.
const (
	StatusPending           TransactionStatus = "pending"
	StatusAuthorized        TransactionStatus = "authorized"
	StatusSucceeded         TransactionStatus = "succeeded"
	StatusFailed            TransactionStatus = "failed"
	StatusCanceled          TransactionStatus = "canceled"
	StatusDisputed          TransactionStatus = "disputed"
	StatusRefunded          TransactionStatus = "refunded"
	StatusPartiallyRefunded TransactionStatus = "partially_refunded"
)

// TransitionEvent descreve a origem de uma transição.
type TransitionEvent struct {
	Source   string     // "webhook", "api", "system"
	SourceID *uuid.UUID // ID do webhook ou requisição (opcional)
	Reason   string     // texto livre opcional
}

// TransitionHistory é o registro imutável de uma transição executada.
type TransitionHistory struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	TransactionID  uuid.UUID
	OldStatus      *TransactionStatus
	NewStatus      TransactionStatus
	EventSource    string
	EventSourceID  *uuid.UUID
	Reason         string
	CreatedAt      time.Time
}

// TransactionForUpdate é o snapshot mínimo retornado pelo SELECT FOR UPDATE.
type TransactionForUpdate struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Status         TransactionStatus
}

// TransactionFSMRepository define as operações de persistência necessárias para a FSM.
type TransactionFSMRepository interface {
	// GetForUpdate busca a transação com SELECT FOR UPDATE dentro de uma tx SQL.
	GetForUpdate(ctx context.Context, organizationID, transactionID uuid.UUID) (*TransactionForUpdate, error)
	// UpdateStatus atualiza o campo status da transação dentro de uma tx SQL.
	UpdateStatus(ctx context.Context, organizationID, transactionID uuid.UUID, newStatus TransactionStatus) error
	// InsertHistory insere um registro em transaction_status_history dentro de uma tx SQL.
	InsertHistory(ctx context.Context, entry TransitionHistory) error
	// ListHistory retorna o histórico de transições ordenado por created_at ASC.
	ListHistory(ctx context.Context, organizationID, transactionID uuid.UUID) ([]TransitionHistory, error)
}

// Erros sentinela da FSM de transações.
var (
	ErrTransactionNotFound = fmt.Errorf("transaction not found")
	ErrTerminalState       = fmt.Errorf("transition from terminal state is not allowed")
	ErrSameStateTransition = fmt.Errorf("transition to the same state is not allowed")
)

// InvalidTransitionError carrega os estados de origem e destino para mensagens descritivas.
type InvalidTransitionError struct {
	From TransactionStatus
	To   TransactionStatus
}

func (e *InvalidTransitionError) Error() string {
	return fmt.Sprintf("invalid transition from %q to %q", e.From, e.To)
}

// ErrInvalidTransition é o sentinel para errors.Is; o erro concreto é *InvalidTransitionError.
var ErrInvalidTransition = fmt.Errorf("invalid transition")

// Is reports whether the target error matches ErrInvalidTransition, enabling errors.Is support.
func (e *InvalidTransitionError) Is(target error) bool {
	return target == ErrInvalidTransition
}
