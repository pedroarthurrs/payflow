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
}

// NewPaymentHandler creates a PaymentHandler.
func NewPaymentHandler(paymentSvc *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentSvc: paymentSvc}
}

// RegisterRoutes registers the payment handler routes.
func (h *PaymentHandler) RegisterRoutes(rg *gin.RouterGroup) {
	payments := rg.Group("/payments")
	payments.POST("", h.Create)
	payments.GET("", h.List)
	payments.GET("/:payment_id", h.GetByID)
}

// createPaymentRequest is the DTO for creating a payment.
type createPaymentRequest struct {
	PaymentMethod string `json:"payment_method" binding:"required,min=1,max=50"`
	AmountCents   int32  `json:"amount_cents"   binding:"required,min=1"`
	// Currency defaults to "BRL" when omitted.
	Currency string `json:"currency" binding:"omitempty,len=3"`
}

type paymentResponse struct {
	ID               uuid.UUID  `json:"id"`
	OrganizationID   uuid.UUID  `json:"organization_id"`
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

	payments, err := h.paymentSvc.List(c.Request.Context(), orgID, pag.ToListParams())
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
