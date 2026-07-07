package domain

import (
	"errors"
	"testing"
)

func TestFSM_ValidTransitions(t *testing.T) {
	fsm := NewStateMachine()
	valid := []struct{ from, to PaymentStatus }{
		{StatusPending, StatusAuthorized},
		{StatusPending, StatusSucceeded},
		{StatusPending, StatusFailed},
		{StatusPending, StatusCanceled},
		{StatusAuthorized, StatusSucceeded},
		{StatusAuthorized, StatusFailed},
		{StatusAuthorized, StatusCanceled},
		{StatusSucceeded, StatusDisputed},
		{StatusSucceeded, StatusRefunded},
		{StatusSucceeded, StatusPartiallyRefunded},
		{StatusFailed, StatusPending},
		{StatusDisputed, StatusSucceeded},
		{StatusDisputed, StatusRefunded},
		{StatusPartiallyRefunded, StatusRefunded},
	}
	for _, tt := range valid {
		if err := fsm.Validate(tt.from, tt.to); err != nil {
			t.Errorf("expected valid transition %s→%s, got: %v", tt.from, tt.to, err)
		}
	}
}

func TestFSM_SameStateTransition(t *testing.T) {
	fsm := NewStateMachine()
	statuses := []PaymentStatus{
		StatusPending, StatusAuthorized, StatusSucceeded, StatusFailed,
		StatusCanceled, StatusDisputed, StatusRefunded, StatusPartiallyRefunded,
	}
	for _, s := range statuses {
		err := fsm.Validate(s, s)
		if !errors.Is(err, ErrSameStateTransition) {
			t.Errorf("expected ErrSameStateTransition for %s→%s, got: %v", s, s, err)
		}
	}
}

func TestFSM_TerminalStateBlocked_Canceled(t *testing.T) {
	fsm := NewStateMachine()
	targets := []PaymentStatus{StatusPending, StatusAuthorized, StatusSucceeded, StatusFailed, StatusDisputed, StatusRefunded, StatusPartiallyRefunded}
	for _, to := range targets {
		err := fsm.Validate(StatusCanceled, to)
		if !errors.Is(err, ErrTerminalState) {
			t.Errorf("expected ErrTerminalState for canceled→%s, got: %v", to, err)
		}
	}
}

func TestFSM_TerminalStateBlocked_Refunded(t *testing.T) {
	fsm := NewStateMachine()
	targets := []PaymentStatus{StatusPending, StatusAuthorized, StatusSucceeded, StatusFailed, StatusCanceled, StatusDisputed, StatusPartiallyRefunded}
	for _, to := range targets {
		err := fsm.Validate(StatusRefunded, to)
		if !errors.Is(err, ErrTerminalState) {
			t.Errorf("expected ErrTerminalState for refunded→%s, got: %v", to, err)
		}
	}
}

func TestFSM_InvalidTransition_FailedToSucceeded(t *testing.T) {
	fsm := NewStateMachine()
	err := fsm.Validate(StatusFailed, StatusSucceeded)
	if !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("expected ErrInvalidTransition for failed→succeeded, got: %v", err)
	}
}

func TestFSM_AllowedTransitions_Terminal_Empty(t *testing.T) {
	fsm := NewStateMachine()
	for _, s := range []PaymentStatus{StatusCanceled, StatusRefunded} {
		allowed := fsm.AllowedTransitions(s)
		if len(allowed) != 0 {
			t.Errorf("expected empty AllowedTransitions for %s, got %v", s, allowed)
		}
	}
}

func TestFSM_IsTerminal_AllStatuses(t *testing.T) {
	fsm := NewStateMachine()
	terminal := map[PaymentStatus]bool{StatusCanceled: true, StatusRefunded: true}
	all := []PaymentStatus{
		StatusPending, StatusAuthorized, StatusSucceeded, StatusFailed,
		StatusCanceled, StatusDisputed, StatusRefunded, StatusPartiallyRefunded,
	}
	for _, s := range all {
		got := fsm.IsTerminal(s)
		want := terminal[s]
		if got != want {
			t.Errorf("IsTerminal(%s): expected %v, got %v", s, want, got)
		}
	}
}

func TestFSM_FailedOnlyToPending(t *testing.T) {
	fsm := NewStateMachine()
	allowed := fsm.AllowedTransitions(StatusFailed)
	if len(allowed) != 1 || allowed[0] != StatusPending {
		t.Errorf("expected failed→[pending] only, got %v", allowed)
	}
	// All other targets must be invalid
	others := []PaymentStatus{StatusAuthorized, StatusSucceeded, StatusCanceled, StatusDisputed, StatusRefunded, StatusPartiallyRefunded}
	for _, to := range others {
		if err := fsm.Validate(StatusFailed, to); err == nil {
			t.Errorf("expected error for failed→%s, got nil", to)
		}
	}
}

func TestFSM_PartiallyRefundedOnlyToRefunded(t *testing.T) {
	fsm := NewStateMachine()
	allowed := fsm.AllowedTransitions(StatusPartiallyRefunded)
	if len(allowed) != 1 || allowed[0] != StatusRefunded {
		t.Errorf("expected partially_refunded→[refunded] only, got %v", allowed)
	}
	others := []PaymentStatus{StatusPending, StatusAuthorized, StatusSucceeded, StatusFailed, StatusCanceled, StatusDisputed}
	for _, to := range others {
		if err := fsm.Validate(StatusPartiallyRefunded, to); err == nil {
			t.Errorf("expected error for partially_refunded→%s, got nil", to)
		}
	}
}
