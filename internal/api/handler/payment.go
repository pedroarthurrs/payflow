package handler

import (
	"errors"
	"net/http"
	"time"

	"payflow/internal/api/dto"
	apierr "payflow/internal/api/err"
	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentHandler handles payment-related HTTP endpoints.
type PaymentHandler struct {
	paymentSvc *service.PaymentService
	gatewaySvc *service.GatewayService
}

// NewPaymentHandler creates a PaymentHandler.
func NewPaymentHandler(paymentSvc *service.PaymentService, gatewaySvc *service.GatewayService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc, gatewaySvc: gatewaySvc}
}

// RegisterRoutes registers the payment handler routes.
func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	payments := rg.Group("/payments")
	payments.POST("", middleware.RequireIdempotency(), h.Create)
	payments.GET("", h.List)
	payments.GET("/:payment_id", h.GetByID)
	payments.GET("/:payment_id/history", h.ListHistory)
	payments.POST("/:payment_id/refund", middleware.RequireRole("member"), h.Refund)
	payments.POST("/:payment_id/cancel", middleware.RequireRole("member"), h.Cancel)
}

// createPaymentRequest is the DTO for creating a payment.
type createPaymentRequest struct {
	CustomerID    uuid.UUID `json:"customer_id"    binding:"required"`
	PaymentMethod string    `json:"payment_method" binding:"required,min=1,max=50"`
	AmountCents   int32     `json:"amount_cents"   binding:"required,min=1"`
	// Currency defaults to "BRL" when omitted.
	Currency string `json:"currency" binding:"omitempty,len=3"`
}

type paymentResponse struct {
	ID               uuid.UUID  `json:"id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
	CustomerID       uuid.UUID  `json:"customer_id"`
	InvoiceID        *uuid.UUID `json:"invoice_id,omitempty"`
	GatewayName      *string    `json:"gateway_name,omitempty"`
	GatewayReference *string    `json:"gateway_reference,omitempty"`
	PaymentMethod    *string    `json:"payment_method,omitempty"`
	AmountCents      int32      `json:"amount_cents"`
	Status           string     `json:"status"`
	Currency         *string    `json:"currency,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

func toPaymentResponse(p *domain.Payment) paymentResponse {
	return paymentResponse{
		ID:               p.ID,
		OrganizationID:   p.OrganizationID,
		CustomerID:       p.CustomerID,
		InvoiceID:        p.InvoiceID,
		GatewayName:      p.GatewayName,
		GatewayReference: p.GatewayReference,
		PaymentMethod:    p.PaymentMethod,
		AmountCents:      p.AmountCents,
		Status:           string(p.Status),
		Currency:         p.Currency,
		CreatedAt:        p.CreatedAt,
		UpdatedAt:        p.UpdatedAt,
	}
}

// transitionHistoryResponse is the DTO for a single payment status transition record.
type transitionHistoryResponse struct {
	ID            uuid.UUID  `json:"id"`
	PaymentID     uuid.UUID  `json:"payment_id"`
	OldStatus     *string    `json:"old_status,omitempty"`
	NewStatus     string     `json:"new_status"`
	EventSource   string     `json:"event_source"`
	EventSourceID *uuid.UUID `json:"event_source_id,omitempty"`
	Reason        string     `json:"reason"`
	CreatedAt     time.Time  `json:"created_at"`
}

func toTransitionHistoryResponse(h domain.TransitionHistory) transitionHistoryResponse {
	r := transitionHistoryResponse{
		ID:            h.ID,
		PaymentID:     h.PaymentID,
		NewStatus:     string(h.NewStatus),
		EventSource:   h.EventSource,
		EventSourceID: h.EventSourceID,
		Reason:        h.Reason,
		CreatedAt:     h.CreatedAt,
	}
	if h.OldStatus != nil {
		s := string(*h.OldStatus)
		r.OldStatus = &s
	}
	return r
}

// Create handles POST /v1/payments — creates the payment then auto-charges via the gateway.
func (h *PaymentHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req createPaymentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	input := domain.PaymentInput{
		CustomerID:    req.CustomerID,
		PaymentMethod: req.PaymentMethod,
		AmountCents:   req.AmountCents,
		Currency:      req.Currency,
	}

	payment, err := h.paymentSvc.Create(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	charged, err := h.gatewaySvc.Charge(c.Request.Context(), orgID, payment.ID, "")
	if err != nil {
		if errors.Is(err, domain.ErrGatewayChargeFailed) {
			_ = c.Error(apierr.PaymentRequired("payment charge failed: " + err.Error()))
			return
		}
		if errors.Is(err, domain.ErrGatewayUnavailable) {
			_ = c.Error(apierr.ServiceUnavailable("payment gateway unavailable, try again later"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toPaymentResponse(charged))
}

// List handles GET /v1/payments.
func (h *PaymentHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	statusStr, ok := dto.ParseStatus(c,
		"pending", "authorized", "succeeded", "failed",
		"canceled", "disputed", "refunded", "partially_refunded",
	)
	if !ok {
		return
	}

	customerIDStr := dto.ParseString(c, "customer_id")
	var customerID *uuid.UUID
	if customerIDStr != nil {
		parsed, err := uuid.Parse(*customerIDStr)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid customer_id: must be a valid UUID"))
			return
		}
		customerID = &parsed
	}

	paymentMethod := dto.ParseString(c, "payment_method")

	dateRange, ok := dto.ParseDateRange(c)
	if !ok {
		return
	}

	params := domain.PaymentListParams{
		ListParams:    pag.ToListParams(),
		CustomerID:    customerID,
		PaymentMethod: paymentMethod,
		CreatedFrom:   dateRange.From,
		CreatedTo:     dateRange.To,
	}
	if statusStr != nil {
		s := domain.PaymentStatus(*statusStr)
		params.Status = &s
	}

	payments, total, err := h.paymentSvc.List(c.Request.Context(), orgID, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]paymentResponse, len(payments))
	for i := range payments {
		resp[i] = toPaymentResponse(&payments[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[paymentResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// GetByID handles GET /v1/payments/:payment_id.
func (h *PaymentHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.PaymentID(c)
	if !ok {
		return
	}

	payment, err := h.paymentSvc.GetByID(c.Request.Context(), orgID, pid)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			_ = c.Error(apierr.NotFound("payment not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toPaymentResponse(payment))
}

// ListHistory handles GET /v1/payments/:payment_id/history.
func (h *PaymentHandler) ListHistory(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.PaymentID(c)
	if !ok {
		return
	}

	history, err := h.paymentSvc.ListHistory(c.Request.Context(), orgID, pid)
	if err != nil {
		if errors.Is(err, domain.ErrPaymentNotFound) {
			_ = c.Error(apierr.NotFound("payment not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	resp := make([]transitionHistoryResponse, len(history))
	for i := range history {
		resp[i] = toTransitionHistoryResponse(history[i])
	}

	c.JSON(http.StatusOK, resp)
}

// gatewayActionRequest is the optional request body for refund and cancel actions.
type gatewayActionRequest struct {
	Reason string `json:"reason"`
}

// mapGatewayError maps gateway service errors to apierr types.
func mapGatewayError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrGatewayChargeFailed),
		errors.Is(err, domain.ErrGatewayRefundFailed),
		errors.Is(err, domain.ErrGatewayCancelFailed):
		_ = c.Error(apierr.PaymentRequired(err.Error()))
	case errors.Is(err, domain.ErrInvalidTransition), errors.Is(err, domain.ErrTerminalState):
		_ = c.Error(apierr.UnprocessableEntity(err.Error()))
	case errors.Is(err, domain.ErrGatewayUnavailable):
		_ = c.Error(apierr.ServiceUnavailable("payment gateway unavailable, try again later"))
	case errors.Is(err, domain.ErrPaymentNotFound):
		_ = c.Error(apierr.NotFound("payment not found"))
	default:
		_ = c.Error(err)
	}
}

// Refund handles POST /v1/payments/:payment_id/refund.
func (h *PaymentHandler) Refund(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.PaymentID(c)
	if !ok {
		return
	}

	var req gatewayActionRequest
	_ = c.ShouldBindJSON(&req) // reason is optional

	payment, err := h.gatewaySvc.Refund(c.Request.Context(), orgID, pid, req.Reason)
	if err != nil {
		mapGatewayError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaymentResponse(payment))
}

// Cancel handles POST /v1/payments/:payment_id/cancel.
func (h *PaymentHandler) Cancel(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pid, ok := dto.PaymentID(c)
	if !ok {
		return
	}

	var req gatewayActionRequest
	_ = c.ShouldBindJSON(&req) // reason is optional

	payment, err := h.gatewaySvc.Cancel(c.Request.Context(), orgID, pid, req.Reason)
	if err != nil {
		mapGatewayError(c, err)
		return
	}

	c.JSON(http.StatusOK, toPaymentResponse(payment))
}
