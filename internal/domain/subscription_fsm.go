package domain

import "fmt"

// SubscriptionStateMachine is the pure FSM for subscriptions — no infrastructure dependencies.
type SubscriptionStateMachine struct{}

// subscriptionTransitionRules is the immutable graph of valid transitions.
var subscriptionTransitionRules = map[SubscriptionStatus][]SubscriptionStatus{
	SubStatusTrialing:   {SubStatusActive, SubStatusCanceled},
	SubStatusActive:     {SubStatusPastDue, SubStatusCanceled, SubStatusPaused},
	SubStatusPastDue:    {SubStatusActive, SubStatusUnpaid, SubStatusCanceled},
	SubStatusUnpaid:     {SubStatusActive, SubStatusCanceled},
	SubStatusPaused:     {SubStatusActive, SubStatusCanceled},
	SubStatusIncomplete: {SubStatusActive, SubStatusCanceled},
	SubStatusCanceled:   {},
}

// subscriptionTerminalStates are irreversible statuses.
var subscriptionTerminalStates = map[SubscriptionStatus]bool{
	SubStatusCanceled: true,
}

// NewSubscriptionStateMachine creates a SubscriptionStateMachine.
func NewSubscriptionStateMachine() *SubscriptionStateMachine {
	return &SubscriptionStateMachine{}
}

// Validate returns nil if the transition from → to is valid, or an error otherwise.
func (sm *SubscriptionStateMachine) Validate(from, to SubscriptionStatus) error {
	if from == to {
		return ErrSameStateTransition
	}
	if subscriptionTerminalStates[from] {
		return ErrSubscriptionCanceled
	}
	for _, allowed := range subscriptionTransitionRules[from] {
		if allowed == to {
			return nil
		}
	}
	return fmt.Errorf("%w: %s → %s", ErrInvalidSubscriptionTransition, from, to)
}

// IsTerminal returns true if status is a terminal state.
func (sm *SubscriptionStateMachine) IsTerminal(status SubscriptionStatus) bool {
	return subscriptionTerminalStates[status]
}
