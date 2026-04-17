package service

import (
	"context"
	"errors"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

var allPaymentStatuses = []domain.PaymentStatus{
	domain.StatusPending, domain.StatusAuthorized, domain.StatusSucceeded,
	domain.StatusFailed, domain.StatusCanceled, domain.StatusDisputed,
	domain.StatusRefunded, domain.StatusPartiallyRefunded,
}

var terminalPaymentStatuses = []domain.PaymentStatus{
	domain.StatusCanceled, domain.StatusRefunded,
}

// TestProperty_FSM_TransitionClassification verifies Property 1: every (from,to) pair falls in exactly one category.
// Feature: critical-tests-coverage, Property 1: FSM classification
func TestProperty_FSM_TransitionClassification(t *testing.T) {
	fsm := domain.NewStateMachine()
	rapid.Check(t, func(t *rapid.T) {
		from := rapid.SampledFrom(allPaymentStatuses).Draw(t, "from")
		to := rapid.SampledFrom(allPaymentStatuses).Draw(t, "to")

		err := fsm.Validate(from, to)

		isValid := err == nil
		isSameState := errors.Is(err, domain.ErrSameStateTransition)
		isTerminal := errors.Is(err, domain.ErrTerminalState)
		isInvalid := errors.Is(err, domain.ErrInvalidTransition)

		categories := 0
		if isValid {
			categories++
		}
		if isSameState {
			categories++
		}
		if isTerminal {
			categories++
		}
		if isInvalid {
			categories++
		}

		if categories != 1 {
			t.Fatalf("pair (%s→%s) fell into %d categories (valid=%v, sameState=%v, terminal=%v, invalid=%v)",
				from, to, categories, isValid, isSameState, isTerminal, isInvalid)
		}
	})
}

// TestProperty_FSM_IsTerminalConsistency verifies Property 2: IsTerminal ↔ AllowedTransitions empty.
// Feature: critical-tests-coverage, Property 2: IsTerminal consistency
func TestProperty_FSM_IsTerminalConsistency(t *testing.T) {
	fsm := domain.NewStateMachine()
	rapid.Check(t, func(t *rapid.T) {
		s := rapid.SampledFrom(allPaymentStatuses).Draw(t, "status")

		isTerminal := fsm.IsTerminal(s)
		allowed := fsm.AllowedTransitions(s)

		if isTerminal && len(allowed) != 0 {
			t.Fatalf("IsTerminal(%s)=true but AllowedTransitions returned %v", s, allowed)
		}
		if !isTerminal && len(allowed) == 0 {
			t.Fatalf("IsTerminal(%s)=false but AllowedTransitions returned empty slice", s)
		}
	})
}

// TestProperty_FSM_Determinism verifies Property 3: Validate is deterministic.
// Feature: critical-tests-coverage, Property 3: FSM determinism
func TestProperty_FSM_Determinism(t *testing.T) {
	fsm := domain.NewStateMachine()
	rapid.Check(t, func(t *rapid.T) {
		from := rapid.SampledFrom(allPaymentStatuses).Draw(t, "from")
		to := rapid.SampledFrom(allPaymentStatuses).Draw(t, "to")

		err1 := fsm.Validate(from, to)
		err2 := fsm.Validate(from, to)

		// Both nil or both non-nil with same sentinel
		if (err1 == nil) != (err2 == nil) {
			t.Fatalf("non-deterministic: first=%v, second=%v", err1, err2)
		}
		if err1 != nil && err2 != nil {
			// Check same sentinel type
			if errors.Is(err1, domain.ErrSameStateTransition) != errors.Is(err2, domain.ErrSameStateTransition) ||
				errors.Is(err1, domain.ErrTerminalState) != errors.Is(err2, domain.ErrTerminalState) ||
				errors.Is(err1, domain.ErrInvalidTransition) != errors.Is(err2, domain.ErrInvalidTransition) {
				t.Fatalf("non-deterministic error type: first=%v, second=%v", err1, err2)
			}
		}
	})
}

// TestProperty_Payment_TransitionHistoryInvariant verifies Property 4: history entries match transitions.
// Feature: critical-tests-coverage, Property 4: transition history invariant
func TestProperty_Payment_TransitionHistoryInvariant(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		paymentID := uuid.New()

		// Build a valid sequence of transitions starting from pending
		// using a small fixed path: pending→authorized→succeeded
		transitions := rapid.SliceOfN(
			rapid.SampledFrom([]domain.PaymentStatus{domain.StatusAuthorized, domain.StatusSucceeded}),
			1, 2,
		).Draw(t, "transitions")

		// Deduplicate to ensure valid sequence: pending→authorized or pending→authorized→succeeded
		var sequence []domain.PaymentStatus
		current := domain.StatusPending
		fsm := domain.NewStateMachine()
		for _, next := range transitions {
			if fsm.Validate(current, next) == nil {
				sequence = append(sequence, next)
				current = next
			}
		}
		if len(sequence) == 0 {
			return // skip if no valid transitions
		}

		var history []domain.TransitionHistory
		currentStatus := domain.StatusPending

		fsmRepo := &mockPaymentFSMRepository{
			getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
				return &domain.PaymentForUpdate{
					ID: paymentID, OrganizationID: orgID, Status: currentStatus,
				}, nil
			},
			insertHistoryFn: func(_ context.Context, entry domain.TransitionHistory) error {
				history = append(history, entry)
				return nil
			},
		}
		svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, nil)

		for _, toStatus := range sequence {
			fsmRepo.getForUpdateFn = func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
				return &domain.PaymentForUpdate{
					ID: paymentID, OrganizationID: orgID, Status: currentStatus,
				}, nil
			}
			if err := svc.Transition(context.Background(), orgID, paymentID, toStatus, domain.TransitionEvent{Source: "test"}); err != nil {
				t.Fatalf("unexpected transition error %s→%s: %v", currentStatus, toStatus, err)
			}
			currentStatus = toStatus
		}

		if len(history) != len(sequence) {
			t.Fatalf("expected %d history entries, got %d", len(sequence), len(history))
		}
		prev := domain.StatusPending
		for i, entry := range history {
			if entry.OldStatus == nil || *entry.OldStatus != prev {
				t.Fatalf("history[%d].OldStatus: expected %s, got %v", i, prev, entry.OldStatus)
			}
			if entry.NewStatus != sequence[i] {
				t.Fatalf("history[%d].NewStatus: expected %s, got %s", i, sequence[i], entry.NewStatus)
			}
			prev = sequence[i]
		}
	})
}

// TestProperty_Payment_MultiTenantIsolation verifies Property 11: wrong org always gets ErrPaymentNotFound.
// Feature: critical-tests-coverage, Property 11: multi-tenant isolation
func TestProperty_Payment_MultiTenantIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate two distinct org UUIDs using random bytes
		orgA := uuid.New()
		orgB := uuid.New()
		for orgB == orgA {
			orgB = uuid.New()
		}
		paymentID := uuid.New()
		_ = rapid.Bool().Draw(t, "seed") // ensure rapid drives the test

		payRepo := &mockPaymentRepository{
			getByIDFn: func(_ context.Context, orgID, _ uuid.UUID) (*domain.Payment, error) {
				if orgID == orgA {
					return &domain.Payment{ID: paymentID, OrganizationID: orgA}, nil
				}
				return nil, domain.ErrPaymentNotFound
			},
		}
		fsmRepo := &mockPaymentFSMRepository{
			getForUpdateFn: func(_ context.Context, orgID, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
				if orgID == orgA {
					return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgA, Status: domain.StatusPending}, nil
				}
				return nil, domain.ErrPaymentNotFound
			},
		}
		svc := newPaymentSvc(fsmRepo, payRepo, nil)

		_, err := svc.GetByID(context.Background(), orgB, paymentID)
		if !errors.Is(err, domain.ErrPaymentNotFound) {
			t.Fatalf("GetByID: expected ErrPaymentNotFound for orgB, got: %v", err)
		}

		err = svc.Transition(context.Background(), orgB, paymentID, domain.StatusAuthorized, domain.TransitionEvent{})
		if !errors.Is(err, domain.ErrPaymentNotFound) {
			t.Fatalf("Transition: expected ErrPaymentNotFound for orgB, got: %v", err)
		}
		if fsmRepo.updateStatusCalled {
			t.Error("UpdateStatus should not be called for wrong org")
		}
	})
}
