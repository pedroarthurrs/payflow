package service

import (
	"context"
	"errors"
	"testing"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// newPaymentSvc builds a PaymentService wiring the given FSM repo into the TxHandle.
func newPaymentSvc(fsmRepo *mockPaymentFSMRepository, payRepo *mockPaymentRepository, ledger paymentLedger) *PaymentService {
	txMgr := &mockTxManager{
		withTxFn: func(ctx context.Context, fn func(tx domain.TxHandle) error) error {
			return fn(&mockTxHandle{paymentFSM: fsmRepo})
		},
	}
	svc := &PaymentService{
		payments: payRepo,
		fsm:      domain.NewStateMachine(),
		txMgr:    txMgr,
		fsmRepo:  fsmRepo,
		ledger:   ledger,
	}
	return svc
}

func pendingPayment(orgID, paymentID uuid.UUID) *domain.PaymentForUpdate {
	return &domain.PaymentForUpdate{
		ID:             paymentID,
		OrganizationID: orgID,
		Status:         domain.StatusPending,
	}
}

// TestPaymentTransition_ValidTransition_CallOrder verifies GetForUpdate → UpdateStatus → InsertHistory order.
func TestPaymentTransition_ValidTransition_CallOrder(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return pendingPayment(orgID, paymentID), nil
		},
	}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, nil)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusAuthorized, domain.TransitionEvent{Source: "test"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	want := []string{"GetForUpdate", "UpdateStatus", "InsertHistory"}
	if len(fsmRepo.callOrder) != len(want) {
		t.Fatalf("expected call order %v, got %v", want, fsmRepo.callOrder)
	}
	for i, call := range want {
		if fsmRepo.callOrder[i] != call {
			t.Errorf("call[%d]: expected %q, got %q", i, call, fsmRepo.callOrder[i])
		}
	}
}

// TestPaymentTransition_GetForUpdateNotFound verifies error propagation without UpdateStatus/InsertHistory.
func TestPaymentTransition_GetForUpdateNotFound(t *testing.T) {
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return nil, domain.ErrPaymentNotFound
		},
	}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, nil)

	err := svc.Transition(context.Background(), uuid.New(), uuid.New(), domain.StatusAuthorized, domain.TransitionEvent{})
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got: %v", err)
	}
	if fsmRepo.updateStatusCalled {
		t.Error("UpdateStatus should not be called when GetForUpdate fails")
	}
	if fsmRepo.insertHistoryCalled {
		t.Error("InsertHistory should not be called when GetForUpdate fails")
	}
}

// TestPaymentTransition_FSMRejectsTerminal verifies ErrTerminalState without UpdateStatus.
func TestPaymentTransition_FSMRejectsTerminal(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{
				ID: paymentID, OrganizationID: orgID, Status: domain.StatusCanceled,
			}, nil
		},
	}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, nil)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusPending, domain.TransitionEvent{})
	if !errors.Is(err, domain.ErrTerminalState) {
		t.Fatalf("expected ErrTerminalState, got: %v", err)
	}
	if fsmRepo.updateStatusCalled {
		t.Error("UpdateStatus should not be called when FSM rejects")
	}
}

// TestPaymentTransition_UpdateStatusFails_NoHistory verifies InsertHistory is not called when UpdateStatus fails.
func TestPaymentTransition_UpdateStatusFails_NoHistory(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	updateErr := errors.New("db error")

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return pendingPayment(orgID, paymentID), nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, _ domain.PaymentStatus) error {
			return updateErr
		},
	}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, nil)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusAuthorized, domain.TransitionEvent{})
	if !errors.Is(err, updateErr) {
		t.Fatalf("expected updateErr, got: %v", err)
	}
	if fsmRepo.insertHistoryCalled {
		t.Error("InsertHistory should not be called when UpdateStatus fails")
	}
}

// TestPaymentTransition_ToSucceeded_LedgerCredit verifies CreditOnPaymentSuccess is called.
func TestPaymentTransition_ToSucceeded_LedgerCredit(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return pendingPayment(orgID, paymentID), nil
		},
	}
	ledger := &mockLedgerService{}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, ledger)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusSucceeded, domain.TransitionEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ledger.creditCalled {
		t.Error("expected CreditOnPaymentSuccess to be called")
	}
}

// TestPaymentTransition_ToSucceeded_LedgerError_ContinuesOK verifies ledger error is non-fatal.
func TestPaymentTransition_ToSucceeded_LedgerError_ContinuesOK(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return pendingPayment(orgID, paymentID), nil
		},
	}
	ledger := &mockLedgerService{
		creditOnPaymentSuccessFn: func(_ context.Context, _ domain.TxHandle, _ uuid.UUID, _ *domain.PaymentForUpdate) error {
			return errors.New("ledger unavailable")
		},
	}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, ledger)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusSucceeded, domain.TransitionEvent{})
	if err != nil {
		t.Fatalf("expected nil (ledger error non-fatal), got: %v", err)
	}
	if !fsmRepo.insertHistoryCalled {
		t.Error("InsertHistory should still be called even when ledger fails")
	}
}

// TestPaymentTransition_ToDisputed_LedgerLock verifies LockOnPaymentDisputed is called.
func TestPaymentTransition_ToDisputed_LedgerLock(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{
				ID: paymentID, OrganizationID: orgID, Status: domain.StatusSucceeded,
			}, nil
		},
	}
	ledger := &mockLedgerService{}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, ledger)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusDisputed, domain.TransitionEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ledger.lockCalled {
		t.Error("expected LockOnPaymentDisputed to be called")
	}
}

// TestPaymentTransition_ToRefunded_LedgerDebit verifies DebitOnPaymentRefund is called.
func TestPaymentTransition_ToRefunded_LedgerDebit(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{
				ID: paymentID, OrganizationID: orgID, Status: domain.StatusSucceeded,
			}, nil
		},
	}
	ledger := &mockLedgerService{}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, ledger)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusRefunded, domain.TransitionEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ledger.debitCalled {
		t.Error("expected DebitOnPaymentRefund to be called")
	}
}

// TestPaymentTransition_DisputedToSucceeded_BothLedgerCalls verifies both credit and release lock are called.
func TestPaymentTransition_DisputedToSucceeded_BothLedgerCalls(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{
				ID: paymentID, OrganizationID: orgID, Status: domain.StatusDisputed,
			}, nil
		},
	}
	ledger := &mockLedgerService{}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, ledger)

	err := svc.Transition(context.Background(), orgID, paymentID, domain.StatusSucceeded, domain.TransitionEvent{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !ledger.creditCalled {
		t.Error("expected CreditOnPaymentSuccess to be called for disputed→succeeded")
	}
	if !ledger.releaseLockCalled {
		t.Error("expected ReleaseLockIfDisputed to be called for disputed→succeeded")
	}
}

// TestPaymentGetByID_WrongOrg_NotFound verifies multi-tenant isolation on GetByID.
func TestPaymentGetByID_WrongOrg_NotFound(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	paymentID := uuid.New()

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, orgID, _ uuid.UUID) (*domain.Payment, error) {
			if orgID == orgA {
				return &domain.Payment{ID: paymentID, OrganizationID: orgA}, nil
			}
			return nil, domain.ErrPaymentNotFound
		},
	}
	svc := newPaymentSvc(&mockPaymentFSMRepository{}, payRepo, nil)

	_, err := svc.GetByID(context.Background(), orgB, paymentID)
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound for wrong org, got: %v", err)
	}
}

// TestPaymentTransition_WrongOrg_NotFound verifies multi-tenant isolation on Transition.
func TestPaymentTransition_WrongOrg_NotFound(t *testing.T) {
	orgA := uuid.New()
	orgB := uuid.New()
	paymentID := uuid.New()

	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, orgID, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			if orgID == orgA {
				return pendingPayment(orgA, paymentID), nil
			}
			return nil, domain.ErrPaymentNotFound
		},
	}
	svc := newPaymentSvc(fsmRepo, &mockPaymentRepository{}, nil)

	err := svc.Transition(context.Background(), orgB, paymentID, domain.StatusAuthorized, domain.TransitionEvent{})
	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound for wrong org, got: %v", err)
	}
	if fsmRepo.updateStatusCalled {
		t.Error("UpdateStatus should not be called for wrong org")
	}
}
