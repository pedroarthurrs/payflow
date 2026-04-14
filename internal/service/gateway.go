package service

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"i9pay/internal/domain"

	"github.com/google/uuid"
)

// GatewayService orchestrates gateway calls with FSM state transitions.
type GatewayService struct {
	paymentSvc *PaymentService
	gateway    domain.Gateway
}

// NewGatewayService creates a GatewayService.
func NewGatewayService(paymentSvc *PaymentService, gateway domain.Gateway) *GatewayService {
	return &GatewayService{
		paymentSvc: paymentSvc,
		gateway:    gateway,
	}
}

// Charge retrieves the payment, calls the gateway, persists the reference, and drives the FSM.
func (s *GatewayService) Charge(ctx context.Context, orgID, paymentID uuid.UUID, reason string) (*domain.Payment, error) {
	payment, err := s.paymentSvc.GetByID(ctx, orgID, paymentID)
	if err != nil {
		return nil, err
	}

	if payment.Status != domain.StatusPending {
		return nil, &domain.InvalidTransitionError{From: payment.Status, To: domain.StatusAuthorized}
	}

	req := buildGatewayRequest(payment)
	result, err := s.gateway.Charge(ctx, req)
	if err != nil {
		if errors.Is(err, domain.ErrGatewayChargeFailed) {
			// Persist error log; log failure but don't block FSM transition
			errLog, _ := json.Marshal(map[string]string{"error": err.Error()})
			if logErr := s.paymentSvc.payments.UpdateErrorLog(ctx, orgID, paymentID, errLog); logErr != nil {
				slog.Error("failed to persist error log after charge failure",
					"organization_id", orgID, "payment_id", paymentID, "error", logErr)
			}
			_ = s.paymentSvc.Transition(ctx, orgID, paymentID, domain.StatusFailed,
				domain.TransitionEvent{Source: "gateway", Reason: reason})
			return nil, err
		}
		// ErrGatewayUnavailable or other — no FSM transition
		return nil, err
	}

	// Persist gateway reference before FSM transition
	if err := s.paymentSvc.payments.UpdateGatewayReference(ctx, orgID, paymentID, result.GatewayName, result.GatewayReference); err != nil {
		return nil, domain.ErrInternal
	}

	if err := s.paymentSvc.Transition(ctx, orgID, paymentID, result.Status,
		domain.TransitionEvent{Source: "gateway", Reason: reason}); err != nil {
		return nil, err
	}

	return s.paymentSvc.GetByID(ctx, orgID, paymentID)
}

// Refund validates the FSM transition, calls the gateway, and drives the FSM to StatusRefunded.
func (s *GatewayService) Refund(ctx context.Context, orgID, paymentID uuid.UUID, reason string) (*domain.Payment, error) {
	payment, err := s.paymentSvc.GetByID(ctx, orgID, paymentID)
	if err != nil {
		return nil, err
	}

	// Pre-validate FSM transition without committing
	allowed := s.paymentSvc.AllowedTransitions(payment.Status)
	if !contains(allowed, domain.StatusRefunded) {
		return nil, &domain.InvalidTransitionError{From: payment.Status, To: domain.StatusRefunded}
	}

	req := buildGatewayRequest(payment)
	if _, err := s.gateway.Refund(ctx, req); err != nil {
		return nil, err
	}

	if err := s.paymentSvc.Transition(ctx, orgID, paymentID, domain.StatusRefunded,
		domain.TransitionEvent{Source: "gateway", Reason: reason}); err != nil {
		return nil, err
	}

	return s.paymentSvc.GetByID(ctx, orgID, paymentID)
}

// Cancel validates the FSM transition, calls the gateway, and drives the FSM to StatusCanceled.
func (s *GatewayService) Cancel(ctx context.Context, orgID, paymentID uuid.UUID, reason string) (*domain.Payment, error) {
	payment, err := s.paymentSvc.GetByID(ctx, orgID, paymentID)
	if err != nil {
		return nil, err
	}

	// Pre-validate FSM transition without committing
	allowed := s.paymentSvc.AllowedTransitions(payment.Status)
	if !contains(allowed, domain.StatusCanceled) {
		return nil, &domain.InvalidTransitionError{From: payment.Status, To: domain.StatusCanceled}
	}

	req := buildGatewayRequest(payment)
	if _, err := s.gateway.Cancel(ctx, req); err != nil {
		return nil, err
	}

	if err := s.paymentSvc.Transition(ctx, orgID, paymentID, domain.StatusCanceled,
		domain.TransitionEvent{Source: "gateway", Reason: reason}); err != nil {
		return nil, err
	}

	return s.paymentSvc.GetByID(ctx, orgID, paymentID)
}

// buildGatewayRequest constructs a GatewayRequest from a domain.Payment.
func buildGatewayRequest(p *domain.Payment) domain.GatewayRequest {
	req := domain.GatewayRequest{
		PaymentID:   p.ID,
		AmountCents: p.AmountCents,
		CustomerID:  p.CustomerID,
	}
	if p.Currency != nil {
		req.Currency = *p.Currency
	}
	if p.PaymentMethod != nil {
		req.PaymentMethod = *p.PaymentMethod
	}
	if p.GatewayReference != nil {
		req.GatewayReference = *p.GatewayReference
	}
	return req
}

// contains reports whether target is in the slice.
func contains(statuses []domain.PaymentStatus, target domain.PaymentStatus) bool {
	for _, s := range statuses {
		if s == target {
			return true
		}
	}
	return false
}
