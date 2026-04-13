package handler

import (
	"errors"
	"net/http"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PaymentHandler handles payment-related HTTP endpoints.
type PaymentHandler struct {
	paymentSvc *service.PaymentService
	fsmSvc     *service.PaymentFSMService
}

// NewPaymentHandler creates a PaymentHandler.
func NewPaymentHandler(paymentSvc *service.PaymentService, fsmSvc *service.PaymentFSMService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc, fsmSvc: fsmSvc}
}

// RegisterRoutes registers the payment handler routes.
func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	payments := rg.Group("/payments")
	payments.POST("", h.Create)
	payments.GET("", h.List)
	payments.GET("/:payment_id", h.GetByID)
	payments.GET("/:payment_id/history", h.ListHistory)
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

// Create handles POST /v1/payments.
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

	c.JSON(http.StatusCreated, toPaymentResponse(payment))
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

	history, err := h.fsmSvc.ListHistory(c.Request.Context(), orgID, pid)
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
