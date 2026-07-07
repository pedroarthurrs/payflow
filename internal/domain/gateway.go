package domain

import (
	"context"

	"github.com/google/uuid"
)

// Gateway is the port that decouples the service layer from any specific payment provider SDK.
type Gateway interface {
	Charge(ctx context.Context, req GatewayRequest) (*ChargeResult, error)
	Refund(ctx context.Context, req GatewayRequest) (*RefundResult, error)
	Cancel(ctx context.Context, req GatewayRequest) (*ChargeResult, error)
}

// GatewayRequest carries all data needed to process a charge, refund, or cancel.
type GatewayRequest struct {
	PaymentID        uuid.UUID
	AmountCents      int32
	Currency         string
	PaymentMethod    string
	CustomerID       uuid.UUID
	GatewayReference string // optional; required for refund/cancel
}

// ChargeResult is the structured response after a charge or cancel attempt.
type ChargeResult struct {
	GatewayReference string
	GatewayName      string
	Status           PaymentStatus
}

// RefundResult is the structured response after a refund attempt.
type RefundResult struct {
	GatewayReference string
	Status           PaymentStatus
}
