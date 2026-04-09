package domain

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// PaymentStatus representa os estados possíveis de um pagamento.
type PaymentStatus string

// Payment status constants representing all possible states of a payment.
const (
	StatusPending           PaymentStatus = "pending"
	StatusAuthorized        PaymentStatus = "authorized"
	StatusSucceeded         PaymentStatus = "succeeded"
	StatusFailed            PaymentStatus = "failed"
	StatusCanceled          PaymentStatus = "canceled"
	StatusDisputed          PaymentStatus = "disputed"
	StatusRefunded          PaymentStatus = "refunded"
	StatusPartiallyRefunded PaymentStatus = "partially_refunded"
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
	PaymentID      uuid.UUID
	OldStatus      *PaymentStatus
	NewStatus      PaymentStatus
	EventSource    string
	EventSourceID  *uuid.UUID
	Reason         string
	CreatedAt      time.Time
}

// PaymentForUpdate é o snapshot mínimo retornado pelo SELECT FOR UPDATE.
type PaymentForUpdate struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Status         PaymentStatus
}

// PaymentFSMRepository define as operações de persistência necessárias para a FSM.
type PaymentFSMRepository interface {
	// GetForUpdate busca o pagamento com SELECT FOR UPDATE dentro de uma tx SQL.
	GetForUpdate(ctx context.Context, organizationID, paymentID uuid.UUID) (*PaymentForUpdate, error)
	// UpdateStatus atualiza o campo status do pagamento dentro de uma tx SQL.
	UpdateStatus(ctx context.Context, organizationID, paymentID uuid.UUID, newStatus PaymentStatus) error
	// InsertHistory insere um registro em payment_status_history dentro de uma tx SQL.
	InsertHistory(ctx context.Context, entry TransitionHistory) error
	// ListHistory retorna o histórico de transições ordenado por created_at ASC.
	ListHistory(ctx context.Context, organizationID, paymentID uuid.UUID) ([]TransitionHistory, error)
}

// Erros sentinela da FSM de pagamentos.
var (
	ErrPaymentNotFound     = fmt.Errorf("payment not found")
	ErrTerminalState       = fmt.Errorf("transition from terminal state is not allowed")
	ErrSameStateTransition = fmt.Errorf("transition to the same state is not allowed")
)

// InvalidTransitionError carrega os estados de origem e destino para mensagens descritivas.
type InvalidTransitionError struct {
	From PaymentStatus
	To   PaymentStatus
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
