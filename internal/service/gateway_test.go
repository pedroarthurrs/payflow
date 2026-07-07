package service

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"payflow/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// newGatewaySvc builds a GatewayService with the given payment repo, FSM repo, and gateway.
func newGatewaySvc(payRepo *mockPaymentRepository, fsmRepo *mockPaymentFSMRepository, gw domain.Gateway) *GatewayService {
	paymentSvc := newPaymentSvc(fsmRepo, payRepo, nil)
	return NewGatewayService(paymentSvc, gw)
}

func pendingPaymentFull(orgID, paymentID uuid.UUID) *domain.Payment {
	return &domain.Payment{
		ID: paymentID, OrganizationID: orgID,
		CustomerID: uuid.New(), AmountCents: 1000,
		Status: domain.StatusPending,
	}
}

func succeededPaymentFull(orgID, paymentID uuid.UUID) *domain.Payment {
	return &domain.Payment{
		ID: paymentID, OrganizationID: orgID,
		CustomerID: uuid.New(), AmountCents: 1000,
		Status: domain.StatusSucceeded,
	}
}

// TestGatewayCharge_Success_PersistsReferenceBeforeFSM verifies UpdateGatewayReference is called before Transition.
func TestGatewayCharge_Success_PersistsReferenceBeforeFSM(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	var callOrder []string

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return pendingPaymentFull(orgID, paymentID), nil
		},
		updateGatewayReferenceFn: func(_ context.Context, _, _ uuid.UUID, _, _ string) error {
			callOrder = append(callOrder, "UpdateGatewayReference")
			return nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			callOrder = append(callOrder, "GetForUpdate")
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusPending}, nil
		},
	}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			return &domain.ChargeResult{GatewayName: "test", GatewayReference: "ref-1", Status: domain.StatusAuthorized}, nil
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, _ = svc.Charge(context.Background(), orgID, paymentID, "test")

	if len(callOrder) < 2 {
		t.Fatalf("expected at least 2 calls, got %v", callOrder)
	}
	if callOrder[0] != "UpdateGatewayReference" {
		t.Errorf("expected UpdateGatewayReference first, got %v", callOrder)
	}
}

// TestGatewayCharge_Success_TransitionsToAuthorized verifies FSM transitions to authorized on success.
func TestGatewayCharge_Success_TransitionsToAuthorized(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	var capturedStatus domain.PaymentStatus

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return pendingPaymentFull(orgID, paymentID), nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusPending}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, s domain.PaymentStatus) error {
			capturedStatus = s
			return nil
		},
	}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			return &domain.ChargeResult{GatewayName: "test", GatewayReference: "ref-1", Status: domain.StatusAuthorized}, nil
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Charge(context.Background(), orgID, paymentID, "test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedStatus != domain.StatusAuthorized {
		t.Errorf("expected transition to authorized, got %s", capturedStatus)
	}
}

// TestGatewayCharge_ChargeFailed_PersistsErrorLog_TransitionsToFailed verifies error log + failed transition.
func TestGatewayCharge_ChargeFailed_PersistsErrorLog_TransitionsToFailed(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	errorLogCalled := false
	var capturedStatus domain.PaymentStatus

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return pendingPaymentFull(orgID, paymentID), nil
		},
		updateErrorLogFn: func(_ context.Context, _, _ uuid.UUID, _ json.RawMessage) error {
			errorLogCalled = true
			return nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusPending}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, s domain.PaymentStatus) error {
			capturedStatus = s
			return nil
		},
	}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			return nil, domain.ErrGatewayChargeFailed
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Charge(context.Background(), orgID, paymentID, "test")

	if !errors.Is(err, domain.ErrGatewayChargeFailed) {
		t.Fatalf("expected ErrGatewayChargeFailed, got: %v", err)
	}
	if !errorLogCalled {
		t.Error("expected UpdateErrorLog to be called")
	}
	if capturedStatus != domain.StatusFailed {
		t.Errorf("expected transition to failed, got %s", capturedStatus)
	}
}

// TestGatewayCharge_GatewayUnavailable_NoFSM verifies no FSM transition on unavailable.
func TestGatewayCharge_GatewayUnavailable_NoFSM(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return pendingPaymentFull(orgID, paymentID), nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			return nil, domain.ErrGatewayUnavailable
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Charge(context.Background(), orgID, paymentID, "test")

	if !errors.Is(err, domain.ErrGatewayUnavailable) {
		t.Fatalf("expected ErrGatewayUnavailable, got: %v", err)
	}
	if fsmRepo.updateStatusCalled {
		t.Error("UpdateStatus should not be called on gateway unavailable")
	}
}

// TestGatewayCharge_NonPendingStatus_NoGatewayCall verifies InvalidTransitionError without calling gateway.
func TestGatewayCharge_NonPendingStatus_NoGatewayCall(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	chargeCalled := false

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: domain.StatusAuthorized}, nil
		},
	}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			chargeCalled = true
			return nil, nil
		},
	}
	svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
	_, err := svc.Charge(context.Background(), orgID, paymentID, "test")

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
	if chargeCalled {
		t.Error("gateway.Charge should not be called for non-pending payment")
	}
}

// TestGatewayCharge_PaymentNotFound verifies propagation without calling gateway.
func TestGatewayCharge_PaymentNotFound(t *testing.T) {
	chargeCalled := false
	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return nil, domain.ErrPaymentNotFound
		},
	}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			chargeCalled = true
			return nil, nil
		},
	}
	svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
	_, err := svc.Charge(context.Background(), uuid.New(), uuid.New(), "test")

	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound, got: %v", err)
	}
	if chargeCalled {
		t.Error("gateway.Charge should not be called when payment not found")
	}
}

// TestGatewayRefund_Succeeded_CallsGatewayAndFSM verifies refund flow from succeeded.
func TestGatewayRefund_Succeeded_CallsGatewayAndFSM(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	refundCalled := false
	var capturedStatus domain.PaymentStatus

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return succeededPaymentFull(orgID, paymentID), nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusSucceeded}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, s domain.PaymentStatus) error {
			capturedStatus = s
			return nil
		},
	}
	gw := &mockGatewayProvider{
		refundFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.RefundResult, error) {
			refundCalled = true
			return &domain.RefundResult{GatewayReference: "ref-1", Status: domain.StatusRefunded}, nil
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Refund(context.Background(), orgID, paymentID, "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !refundCalled {
		t.Error("expected gateway.Refund to be called")
	}
	if capturedStatus != domain.StatusRefunded {
		t.Errorf("expected transition to refunded, got %s", capturedStatus)
	}
}

// TestGatewayRefund_InvalidStatus_NoGatewayCall verifies InvalidTransitionError without calling gateway.
func TestGatewayRefund_InvalidStatus_NoGatewayCall(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	refundCalled := false

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: domain.StatusPending}, nil
		},
	}
	gw := &mockGatewayProvider{
		refundFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.RefundResult, error) {
			refundCalled = true
			return nil, nil
		},
	}
	svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
	_, err := svc.Refund(context.Background(), orgID, paymentID, "test")

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
	if refundCalled {
		t.Error("gateway.Refund should not be called for invalid status")
	}
}

// TestGatewayRefund_GatewayError_NoFSM verifies no FSM transition when gateway.Refund fails.
func TestGatewayRefund_GatewayError_NoFSM(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return succeededPaymentFull(orgID, paymentID), nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{}
	gw := &mockGatewayProvider{
		refundFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.RefundResult, error) {
			return nil, domain.ErrGatewayRefundFailed
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Refund(context.Background(), orgID, paymentID, "test")

	if err == nil {
		t.Fatal("expected error from gateway.Refund")
	}
	if fsmRepo.updateStatusCalled {
		t.Error("UpdateStatus should not be called when gateway.Refund fails")
	}
}

// TestGatewayCancel_Pending_CallsGatewayAndFSM verifies cancel flow from pending.
func TestGatewayCancel_Pending_CallsGatewayAndFSM(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	cancelCalled := false
	var capturedStatus domain.PaymentStatus

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return pendingPaymentFull(orgID, paymentID), nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusPending}, nil
		},
		updateStatusFn: func(_ context.Context, _, _ uuid.UUID, s domain.PaymentStatus) error {
			capturedStatus = s
			return nil
		},
	}
	gw := &mockGatewayProvider{
		cancelFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			cancelCalled = true
			return &domain.ChargeResult{GatewayName: "test", GatewayReference: "ref-1", Status: domain.StatusCanceled}, nil
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Cancel(context.Background(), orgID, paymentID, "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !cancelCalled {
		t.Error("expected gateway.Cancel to be called")
	}
	if capturedStatus != domain.StatusCanceled {
		t.Errorf("expected transition to canceled, got %s", capturedStatus)
	}
}

// TestGatewayCancel_TerminalStatus_NoGatewayCall verifies InvalidTransitionError for terminal status.
func TestGatewayCancel_TerminalStatus_NoGatewayCall(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	cancelCalled := false

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: domain.StatusCanceled}, nil
		},
	}
	gw := &mockGatewayProvider{
		cancelFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			cancelCalled = true
			return nil, nil
		},
	}
	svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
	_, err := svc.Cancel(context.Background(), orgID, paymentID, "test")

	if !errors.Is(err, domain.ErrInvalidTransition) {
		t.Fatalf("expected ErrInvalidTransition, got: %v", err)
	}
	if cancelCalled {
		t.Error("gateway.Cancel should not be called for terminal status")
	}
}

// TestGatewayCancel_GatewayError_NoFSM verifies no FSM transition when gateway.Cancel fails.
func TestGatewayCancel_GatewayError_NoFSM(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return pendingPaymentFull(orgID, paymentID), nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{}
	gw := &mockGatewayProvider{
		cancelFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			return nil, domain.ErrGatewayCancelFailed
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	_, err := svc.Cancel(context.Background(), orgID, paymentID, "test")

	if err == nil {
		t.Fatal("expected error from gateway.Cancel")
	}
	if fsmRepo.updateStatusCalled {
		t.Error("UpdateStatus should not be called when gateway.Cancel fails")
	}
}

// TestGatewayRefund_Success_ReturnsUpdatedPayment verifies Refund returns updated payment via GetByID.
func TestGatewayRefund_Success_ReturnsUpdatedPayment(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	getByIDCallCount := 0

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			getByIDCallCount++
			status := domain.StatusSucceeded
			if getByIDCallCount > 1 {
				status = domain.StatusRefunded
			}
			return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: status}, nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusSucceeded}, nil
		},
	}
	gw := &mockGatewayProvider{
		refundFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.RefundResult, error) {
			return &domain.RefundResult{GatewayReference: "ref-1", Status: domain.StatusRefunded}, nil
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	result, err := svc.Refund(context.Background(), orgID, paymentID, "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
		return
	}
	if result == nil {
		t.Fatal("expected non-nil result")
		return
	}
	if result.Status != domain.StatusRefunded {
		t.Errorf("expected refunded status in result, got %s", result.Status)
	}
}

// TestGatewayCancel_Success_ReturnsUpdatedPayment verifies Cancel returns updated payment via GetByID.
func TestGatewayCancel_Success_ReturnsUpdatedPayment(t *testing.T) {
	orgID := uuid.New()
	paymentID := uuid.New()
	getByIDCallCount := 0

	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			getByIDCallCount++
			status := domain.StatusPending
			if getByIDCallCount > 1 {
				status = domain.StatusCanceled
			}
			return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: status}, nil
		},
	}
	fsmRepo := &mockPaymentFSMRepository{
		getForUpdateFn: func(_ context.Context, _, _ uuid.UUID) (*domain.PaymentForUpdate, error) {
			return &domain.PaymentForUpdate{ID: paymentID, OrganizationID: orgID, Status: domain.StatusPending}, nil
		},
	}
	gw := &mockGatewayProvider{
		cancelFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			return &domain.ChargeResult{GatewayName: "test", GatewayReference: "ref-1", Status: domain.StatusCanceled}, nil
		},
	}
	svc := newGatewaySvc(payRepo, fsmRepo, gw)
	result, err := svc.Cancel(context.Background(), orgID, paymentID, "test")

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result == nil || result.Status != domain.StatusCanceled {
		t.Errorf("expected canceled status in result, got %v", result)
	}
}

// TestGatewayCharge_WrongOrg_NotFound verifies multi-tenant isolation on Charge.
func TestGatewayCharge_WrongOrg_NotFound(t *testing.T) {
	chargeCalled := false
	payRepo := &mockPaymentRepository{
		getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
			return nil, domain.ErrPaymentNotFound
		},
	}
	gw := &mockGatewayProvider{
		chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
			chargeCalled = true
			return nil, nil
		},
	}
	svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
	_, err := svc.Charge(context.Background(), uuid.New(), uuid.New(), "test")

	if !errors.Is(err, domain.ErrPaymentNotFound) {
		t.Fatalf("expected ErrPaymentNotFound for wrong org, got: %v", err)
	}
	if chargeCalled {
		t.Error("gateway.Charge should not be called for wrong org")
	}
}

// TestProperty_Gateway_ChargeRejectsNonPending verifies Property 8: Charge rejects non-pending without calling gateway.
// Feature: critical-tests-coverage, Property 8: charge rejects non-pending
func TestProperty_Gateway_ChargeRejectsNonPending(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom([]domain.PaymentStatus{
			domain.StatusAuthorized, domain.StatusSucceeded, domain.StatusFailed,
			domain.StatusCanceled, domain.StatusDisputed, domain.StatusRefunded,
			domain.StatusPartiallyRefunded,
		}).Draw(t, "status")

		orgID := uuid.New()
		paymentID := uuid.New()
		chargeCalled := false

		payRepo := &mockPaymentRepository{
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
				return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: status}, nil
			},
		}
		gw := &mockGatewayProvider{
			chargeFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
				chargeCalled = true
				return nil, nil
			},
		}
		svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
		_, err := svc.Charge(context.Background(), orgID, paymentID, "test")

		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("status=%s: expected ErrInvalidTransition, got: %v", status, err)
		}
		if chargeCalled {
			t.Errorf("status=%s: gateway.Charge should not be called", status)
		}
	})
}

// TestProperty_Gateway_RefundRejectsInvalidStatus verifies Property 9: Refund rejects statuses that can't transition to refunded.
// Feature: critical-tests-coverage, Property 9: refund rejects invalid status
func TestProperty_Gateway_RefundRejectsInvalidStatus(t *testing.T) {
	fsm := domain.NewStateMachine()
	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom(allPaymentStatuses).Draw(t, "status")
		if contains(fsm.AllowedTransitions(status), domain.StatusRefunded) {
			return // skip valid statuses
		}

		orgID := uuid.New()
		paymentID := uuid.New()
		refundCalled := false

		payRepo := &mockPaymentRepository{
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
				return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: status}, nil
			},
		}
		gw := &mockGatewayProvider{
			refundFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.RefundResult, error) {
				refundCalled = true
				return nil, nil
			},
		}
		svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
		_, err := svc.Refund(context.Background(), orgID, paymentID, "test")

		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("status=%s: expected ErrInvalidTransition, got: %v", status, err)
		}
		if refundCalled {
			t.Errorf("status=%s: gateway.Refund should not be called", status)
		}
	})
}

// TestProperty_Gateway_CancelRejectsTerminal verifies Property 10: Cancel rejects terminal statuses.
// Feature: critical-tests-coverage, Property 10: cancel rejects terminal
func TestProperty_Gateway_CancelRejectsTerminal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		status := rapid.SampledFrom(terminalPaymentStatuses).Draw(t, "status")

		orgID := uuid.New()
		paymentID := uuid.New()
		cancelCalled := false

		payRepo := &mockPaymentRepository{
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.Payment, error) {
				return &domain.Payment{ID: paymentID, OrganizationID: orgID, Status: status}, nil
			},
		}
		gw := &mockGatewayProvider{
			cancelFn: func(_ context.Context, _ domain.GatewayRequest) (*domain.ChargeResult, error) {
				cancelCalled = true
				return nil, nil
			},
		}
		svc := newGatewaySvc(payRepo, &mockPaymentFSMRepository{}, gw)
		_, err := svc.Cancel(context.Background(), orgID, paymentID, "test")

		if !errors.Is(err, domain.ErrInvalidTransition) {
			t.Fatalf("status=%s: expected ErrInvalidTransition, got: %v", status, err)
		}
		if cancelCalled {
			t.Errorf("status=%s: gateway.Cancel should not be called", status)
		}
	})
}
