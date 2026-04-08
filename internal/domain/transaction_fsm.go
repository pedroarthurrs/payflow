package domain

// StateMachine é a FSM pura de transações — sem dependências de infraestrutura.
type StateMachine struct{}

// transitionRules é o grafo imutável de transições válidas, inicializado uma única vez.
var transitionRules = map[TransactionStatus][]TransactionStatus{
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
var terminalStates = map[TransactionStatus]bool{
	StatusCanceled: true,
	StatusRefunded: true,
}

// NewStateMachine cria uma StateMachine.
func NewStateMachine() *StateMachine { return &StateMachine{} }

// Validate verifica se a transição de from para to é permitida.
// Retorna nil se válida, ou um dos erros sentinela caso contrário.
func (sm *StateMachine) Validate(from, to TransactionStatus) error {
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
func (sm *StateMachine) AllowedTransitions(from TransactionStatus) []TransactionStatus {
	return transitionRules[from]
}

// IsTerminal retorna true se status é um estado terminal (canceled ou refunded).
func (sm *StateMachine) IsTerminal(status TransactionStatus) bool {
	return terminalStates[status]
}
