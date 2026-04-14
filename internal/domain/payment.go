package domain

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// Payment represents a payment entity in the domain layer.
type Payment struct {
	ID               uuid.UUID
	OrganizationID   uuid.UUID
	CustomerID       uuid.UUID
	InvoiceID        *uuid.UUID
	SubscriptionID   *uuid.UUID
	GatewayName      *string
	GatewayReference *string
	PaymentMethod    *string
	AmountCents      int32
	Status           PaymentStatus
	Currency         *string
	IdempotencyKey   *string
	ErrorLog         json.RawMessage
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

// PaymentInput represents the input for creating a payment.
type PaymentInput struct {
	CustomerID     uuid.UUID
	SubscriptionID *uuid.UUID
	PaymentMethod  string
	AmountCents    int32
	// Currency defaults to "BRL" when empty.
	Currency       string
	IdempotencyKey string
}

// PaymentListParams carries optional filters for listing payments.
type PaymentListParams struct {
	ListParams
	CustomerID    *uuid.UUID
	Status        *PaymentStatus
	PaymentMethod *string
	CreatedFrom   *time.Time
	CreatedTo     *time.Time
}

// PaymentRepository defines persistence operations for the payment domain.
type PaymentRepository interface {
	Create(ctx context.Context, organizationID uuid.UUID, input PaymentInput) (*Payment, error)
	GetByID(ctx context.Context, organizationID, paymentID uuid.UUID) (*Payment, error)
	List(ctx context.Context, organizationID uuid.UUID, params PaymentListParams) ([]Payment, error)
	Count(ctx context.Context, organizationID uuid.UUID, params PaymentListParams) (int64, error)
	UpdateGatewayReference(ctx context.Context, organizationID, paymentID uuid.UUID, gatewayName, gatewayReference string) error
	UpdateErrorLog(ctx context.Context, organizationID, paymentID uuid.UUID, errorLog json.RawMessage) error
}

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

// StateMachine é a FSM pura de pagamentos — sem dependências de infraestrutura.
type StateMachine struct{}

// transitionRules é o grafo imutável de transições válidas, inicializado uma única vez.
var transitionRules = map[PaymentStatus][]PaymentStatus{
	StatusPending:           {StatusAuthorized, StatusSucceeded, StatusFailed, StatusCanceled},
	StatusAuthorized:        {StatusSucceeded, StatusFailed, StatusCanceled},
	StatusSucceeded:         {StatusDisputed, StatusRefunded, StatusPartiallyRefunded},
	StatusFailed:            {StatusPending},
	StatusCanceled:          {},
	StatusDisputed:          {StatusSucceeded, StatusRefunded},
	StatusRefunded:          {},
	StatusPartiallyRefunded: {StatusRefunded},
}

// terminalStates é o conjunto de estados dos quais nenhuma transição é permitida.
var terminalStates = map[PaymentStatus]bool{
	StatusCanceled: true,
	StatusRefunded: true,
}

// NewStateMachine cria uma StateMachine.
func NewStateMachine() *StateMachine { return &StateMachine{} }

// Validate verifica se a transição de from para to é permitida.
// Retorna nil se válida, ou um dos erros sentinela caso contrário.
func (sm *StateMachine) Validate(from, to PaymentStatus) error {
	if from == to {
		return ErrSameStateTransition
	}
	if terminalStates[from] {
		return ErrTerminalState
	}
	for _, allowed := range transitionRules[from] {
		if allowed == to {
			return nil
		}
	}
	return &InvalidTransitionError{From: from, To: to}
}

// AllowedTransitions retorna os estados de destino permitidos a partir de from.
// Retorna slice vazia para estados terminais.
func (sm *StateMachine) AllowedTransitions(from PaymentStatus) []PaymentStatus {
	return transitionRules[from]
}

// IsTerminal retorna true se status é um estado terminal (canceled ou refunded).
func (sm *StateMachine) IsTerminal(status PaymentStatus) bool {
	return terminalStates[status]
}
