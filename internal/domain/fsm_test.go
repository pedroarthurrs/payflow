package domain

import (
	"errors"
	"testing"
)

// ── SubscriptionStateMachine Tests ──────────────────────────────────────────

func TestSubscriptionFSM_ValidTransitions(t *testing.T) {
	fsm := NewSubscriptionStateMachine()

	validTransitions := []struct {
		from SubscriptionStatus
		to   SubscriptionStatus
	}{
		{SubStatusTrialing, SubStatusActive},
		{SubStatusTrialing, SubStatusCanceled},
		{SubStatusActive, SubStatusPastDue},
		{SubStatusActive, SubStatusCanceled},
		{SubStatusActive, SubStatusPaused},
		{SubStatusPastDue, SubStatusActive},
		{SubStatusPastDue, SubStatusUnpaid},
		{SubStatusPastDue, SubStatusCanceled},
		{SubStatusUnpaid, SubStatusActive},
		{SubStatusUnpaid, SubStatusCanceled},
		{SubStatusPaused, SubStatusActive},
		{SubStatusPaused, SubStatusCanceled},
		{SubStatusIncomplete, SubStatusActive},
		{SubStatusIncomplete, SubStatusCanceled},
	}

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			if err := fsm.Validate(tt.from, tt.to); err != nil {
				t.Errorf("expected valid transition %s → %s, got error: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestSubscriptionFSM_InvalidTransitions(t *testing.T) {
	fsm := NewSubscriptionStateMachine()

	invalidTransitions := []struct {
		from SubscriptionStatus
		to   SubscriptionStatus
	}{
		// Canceled is terminal
		{SubStatusCanceled, SubStatusActive},
		{SubStatusCanceled, SubStatusPaused},
		{SubStatusCanceled, SubStatusTrialing},

		// Invalid state transitions
		{SubStatusTrialing, SubStatusPastDue},
		{SubStatusTrialing, SubStatusUnpaid},
		{SubStatusTrialing, SubStatusPaused},
		{SubStatusActive, SubStatusTrialing},
		{SubStatusActive, SubStatusUnpaid},
		{SubStatusActive, SubStatusIncomplete},
		{SubStatusPastDue, SubStatusPaused},
		{SubStatusPastDue, SubStatusTrialing},
		{SubStatusUnpaid, SubStatusPaused},
		{SubStatusUnpaid, SubStatusTrialing},
		{SubStatusPaused, SubStatusPastDue},
		{SubStatusPaused, SubStatusUnpaid},
		{SubStatusIncomplete, SubStatusPaused},
		{SubStatusIncomplete, SubStatusPastDue},
	}

	for _, tt := range invalidTransitions {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			if err := fsm.Validate(tt.from, tt.to); err == nil {
				t.Errorf("expected error for invalid transition %s → %s, got nil", tt.from, tt.to)
			}
		})
	}
}

func TestSubscriptionFSM_SameState(t *testing.T) {
	fsm := NewSubscriptionStateMachine()

	states := []SubscriptionStatus{
		SubStatusTrialing, SubStatusActive, SubStatusPastDue,
		SubStatusUnpaid, SubStatusCanceled, SubStatusPaused, SubStatusIncomplete,
	}

	for _, s := range states {
		t.Run("same_"+string(s), func(t *testing.T) {
			err := fsm.Validate(s, s)
			if !errors.Is(err, ErrSameStateTransition) {
				t.Errorf("expected ErrSameStateTransition for %s → %s, got %v", s, s, err)
			}
		})
	}
}

func TestSubscriptionFSM_IsTerminal(t *testing.T) {
	fsm := NewSubscriptionStateMachine()

	if !fsm.IsTerminal(SubStatusCanceled) {
		t.Error("expected canceled to be terminal")
	}

	nonTerminal := []SubscriptionStatus{
		SubStatusTrialing, SubStatusActive, SubStatusPastDue,
		SubStatusUnpaid, SubStatusPaused, SubStatusIncomplete,
	}
	for _, s := range nonTerminal {
		if fsm.IsTerminal(s) {
			t.Errorf("expected %s to NOT be terminal", s)
		}
	}
}

func TestSubscriptionFSM_CanceledFromTerminal(t *testing.T) {
	fsm := NewSubscriptionStateMachine()

	err := fsm.Validate(SubStatusCanceled, SubStatusActive)
	if !errors.Is(err, ErrSubscriptionCanceled) {
		t.Errorf("expected ErrSubscriptionCanceled, got %v", err)
	}
}

// ── Payment FSM Edge Cases ──────────────────────────────────────────────────

func TestPaymentFSM_ValidTransitions(t *testing.T) {
	fsm := NewStateMachine()

	validTransitions := []struct {
		from PaymentStatus
		to   PaymentStatus
	}{
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

	for _, tt := range validTransitions {
		t.Run(string(tt.from)+"→"+string(tt.to), func(t *testing.T) {
			if err := fsm.Validate(tt.from, tt.to); err != nil {
				t.Errorf("expected valid transition %s → %s, got error: %v", tt.from, tt.to, err)
			}
		})
	}
}

func TestPaymentFSM_TerminalStates(t *testing.T) {
	fsm := NewStateMachine()

	// Canceled and Refunded are terminal
	terminalStates := []PaymentStatus{StatusCanceled, StatusRefunded}

	for _, s := range terminalStates {
		if !fsm.IsTerminal(s) {
			t.Errorf("expected %s to be terminal", s)
		}
		err := fsm.Validate(s, StatusPending)
		if !errors.Is(err, ErrTerminalState) {
			t.Errorf("expected ErrTerminalState for %s → pending, got %v", s, err)
		}
	}
}

func TestPaymentFSM_FailedCanRetry(t *testing.T) {
	fsm := NewStateMachine()

	// Failed → Pending is allowed (retry)
	if err := fsm.Validate(StatusFailed, StatusPending); err != nil {
		t.Errorf("expected failed → pending (retry) to be valid, got %v", err)
	}

	// But Failed → Succeeded is not (must go through Pending first)
	if err := fsm.Validate(StatusFailed, StatusSucceeded); err == nil {
		t.Error("expected failed → succeeded to be invalid")
	}
}

func TestPaymentFSM_DisputeResolutions(t *testing.T) {
	fsm := NewStateMachine()

	// Disputed can resolve to Succeeded (won dispute) or Refunded (lost)
	if err := fsm.Validate(StatusDisputed, StatusSucceeded); err != nil {
		t.Errorf("expected disputed → succeeded, got %v", err)
	}
	if err := fsm.Validate(StatusDisputed, StatusRefunded); err != nil {
		t.Errorf("expected disputed → refunded, got %v", err)
	}

	// But Disputed → Pending is not valid
	if err := fsm.Validate(StatusDisputed, StatusPending); err == nil {
		t.Error("expected disputed → pending to be invalid")
	}
}

func TestPaymentFSM_PartialRefundToFull(t *testing.T) {
	fsm := NewStateMachine()

	// PartiallyRefunded can go to Refunded only
	if err := fsm.Validate(StatusPartiallyRefunded, StatusRefunded); err != nil {
		t.Errorf("expected partially_refunded → refunded, got %v", err)
	}

	// Can't go to anything else
	others := []PaymentStatus{StatusPending, StatusAuthorized, StatusSucceeded, StatusFailed, StatusCanceled}
	for _, s := range others {
		if err := fsm.Validate(StatusPartiallyRefunded, s); err == nil {
			t.Errorf("expected partially_refunded → %s to be invalid", s)
		}
	}
}

func TestPaymentFSM_InvalidTransitionError(t *testing.T) {
	fsm := NewStateMachine()

	err := fsm.Validate(StatusPending, StatusRefunded)
	if err == nil {
		t.Fatal("expected error")
	}

	var ite *InvalidTransitionError
	if !errors.As(err, &ite) {
		t.Fatalf("expected *InvalidTransitionError, got %T", err)
	}
	if ite.From != StatusPending || ite.To != StatusRefunded {
		t.Errorf("expected from=pending to=refunded, got from=%s to=%s", ite.From, ite.To)
	}

	// errors.Is support
	if !errors.Is(err, ErrInvalidTransition) {
		t.Error("expected errors.Is(err, ErrInvalidTransition) to be true")
	}
}

func TestPaymentFSM_AllowedTransitions(t *testing.T) {
	fsm := NewStateMachine()

	// Canceled should have no allowed transitions
	allowed := fsm.AllowedTransitions(StatusCanceled)
	if len(allowed) != 0 {
		t.Errorf("expected 0 transitions from canceled, got %d", len(allowed))
	}

	// Pending should have 4 options
	allowed = fsm.AllowedTransitions(StatusPending)
	if len(allowed) != 4 {
		t.Errorf("expected 4 transitions from pending, got %d", len(allowed))
	}
}
