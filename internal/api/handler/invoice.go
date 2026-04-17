package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// InvoiceHandler handles invoice and invoice-settings HTTP endpoints.
type InvoiceHandler struct {
	svc     *service.InvoiceService
	permSvc middleware.PermissionResolver
}

// NewInvoiceHandler creates an InvoiceHandler.
func NewInvoiceHandler(svc *service.InvoiceService, permSvc middleware.PermissionResolver) *InvoiceHandler {
	return &InvoiceHandler{svc: svc, permSvc: permSvc}
}

// RegisterRoutes registers all invoice and invoice-settings routes.
func (h *InvoiceHandler) RegisterRoutes(rg *gin.RouterGroup) {
	invoices := rg.Group("/invoices")
	invoices.POST("", middleware.RequirePermission(h.permSvc, domain.PermInvoicesWrite), middleware.RequireIdempotency(), h.Create)
	invoices.GET("", middleware.RequirePermission(h.permSvc, domain.PermInvoicesRead), h.List)
	invoices.GET("/:invoice_id", middleware.RequirePermission(h.permSvc, domain.PermInvoicesRead), h.GetByID)
	invoices.POST("/:invoice_id/cancel", middleware.RequirePermission(h.permSvc, domain.PermInvoicesWrite), h.Cancel)
	invoices.POST("/:invoice_id/transition", middleware.RequirePermission(h.permSvc, domain.PermInvoicesWrite), h.Transition)
	invoices.GET("/:invoice_id/logs", middleware.RequirePermission(h.permSvc, domain.PermInvoicesRead), h.ListLogs)

	settings := rg.Group("/invoice-settings")
	settings.PUT("/:type", middleware.RequirePermission(h.permSvc, domain.PermInvoicesManage), h.UpsertSetting)
	settings.GET("", middleware.RequirePermission(h.permSvc, domain.PermInvoicesRead), h.ListSettings)
	settings.GET("/:type", middleware.RequirePermission(h.permSvc, domain.PermInvoicesRead), h.GetSetting)
	settings.DELETE("/:type", middleware.RequirePermission(h.permSvc, domain.PermInvoicesManage), h.DeleteSetting)
}

// =============================================================================
// Request / Response DTOs
// =============================================================================

type createInvoiceItemRequest struct {
	Description    string `json:"description"      binding:"required"`
	Quantity       string `json:"quantity"          binding:"required"`
	UnitPriceCents int32  `json:"unit_price_cents"  binding:"required,min=0"`
}

type createInvoiceRequest struct {
	CustomerID  uuid.UUID                  `json:"customer_id"  binding:"required"`
	InvoiceType string                     `json:"invoice_type" binding:"required,oneof=national international"`
	Items       []createInvoiceItemRequest `json:"items"        binding:"required,min=1"`
}

type transitionInvoiceRequest struct {
	Status  string `json:"status"  binding:"required"`
	Message string `json:"message" binding:"omitempty"`
}

type upsertInvoiceSettingRequest struct {
	ERPApiKey           *string `json:"erp_api_key"            binding:"omitempty"`
	ERPApiSecret        *string `json:"erp_api_secret"         binding:"omitempty"`
	ERPAccountID        *string `json:"erp_account_id"         binding:"omitempty"`
	NfseServiceCode     *string `json:"nfse_service_code"      binding:"omitempty"`
	NfseItemListService *string `json:"nfse_item_list_service" binding:"omitempty"`
	TaxRate             *string `json:"tax_rate"               binding:"omitempty"`
	InvoiceSeries       *string `json:"invoice_series"         binding:"omitempty"`
	IsoCurrency         *string `json:"iso_currency"           binding:"omitempty"`
	ExportRegime        *string `json:"export_regime"          binding:"omitempty"`
	IssueOn             string  `json:"issue_on"               binding:"omitempty"`
	IsActive            *bool   `json:"is_active"              binding:"omitempty"`
}

type invoiceItemResponse struct {
	ID              uuid.UUID       `json:"id"`
	OrganizationID  uuid.UUID       `json:"organization_id"`
	InvoiceID       uuid.UUID       `json:"invoice_id"`
	Description     string          `json:"description"`
	Quantity        string          `json:"quantity"`
	UnitPriceCents  int32           `json:"unit_price_cents"`
	TotalPriceCents int32           `json:"total_price_cents"`
	Metadata        json.RawMessage `json:"metadata,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type invoiceResponse struct {
	ID               uuid.UUID             `json:"id"`
	OrganizationID   uuid.UUID             `json:"organization_id"`
	CustomerID       uuid.UUID             `json:"customer_id"`
	SubscriptionID   *uuid.UUID            `json:"subscription_id,omitempty"`
	InvoiceType      string                `json:"invoice_type"`
	OperationType    int32                 `json:"operation_type"`
	AmountCents      int32                 `json:"amount_cents"`
	Currency         string                `json:"currency"`
	Status           string                `json:"status"`
	ERPReference     *string               `json:"erp_reference,omitempty"`
	CustomerSnapshot json.RawMessage       `json:"customer_snapshot,omitempty"`
	ItemsSnapshot    json.RawMessage       `json:"items_snapshot,omitempty"`
	LastSyncLog      json.RawMessage       `json:"last_sync_log,omitempty"`
	Items            []invoiceItemResponse `json:"items,omitempty"`
	CreatedAt        time.Time             `json:"created_at"`
	UpdatedAt        time.Time             `json:"updated_at"`
}

type invoiceLogResponse struct {
	ID              uuid.UUID       `json:"id"`
	OrganizationID  uuid.UUID       `json:"organization_id"`
	InvoiceID       uuid.UUID       `json:"invoice_id"`
	EventType       string          `json:"event_type"`
	InvoiceStatus   *string         `json:"invoice_status,omitempty"`
	ERPReference    *string         `json:"erp_reference,omitempty"`
	ERPErrorCode    *string         `json:"erp_error_code,omitempty"`
	ERPErrorMessage *string         `json:"erp_error_message,omitempty"`
	HTTPStatus      *int32          `json:"http_status,omitempty"`
	RequestURL      *string         `json:"request_url,omitempty"`
	RequestPayload  json.RawMessage `json:"request_payload,omitempty"`
	ResponseBody    json.RawMessage `json:"response_body,omitempty"`
	AttemptNumber   int32           `json:"attempt_number"`
	Success         bool            `json:"success"`
	Message         string          `json:"message"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type invoiceSettingResponse struct {
	ID                  uuid.UUID `json:"id"`
	OrganizationID      uuid.UUID `json:"organization_id"`
	Type                string    `json:"type"`
	ERPApiKey           *string   `json:"erp_api_key,omitempty"`
	ERPApiSecret        *string   `json:"erp_api_secret,omitempty"`
	ERPAccountID        *string   `json:"erp_account_id,omitempty"`
	NfseServiceCode     *string   `json:"nfse_service_code,omitempty"`
	NfseItemListService *string   `json:"nfse_item_list_service,omitempty"`
	TaxRate             *string   `json:"tax_rate,omitempty"`
	InvoiceSeries       *string   `json:"invoice_series,omitempty"`
	IsoCurrency         *string   `json:"iso_currency,omitempty"`
	ExportRegime        *string   `json:"export_regime,omitempty"`
	IssueOn             string    `json:"issue_on"`
	IsActive            bool      `json:"is_active"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// =============================================================================
// Converters
// =============================================================================

func toInvoiceResponse(inv *domain.Invoice) invoiceResponse {
	resp := invoiceResponse{
		ID:               inv.ID,
		OrganizationID:   inv.OrganizationID,
		CustomerID:       inv.CustomerID,
		SubscriptionID:   inv.SubscriptionID,
		InvoiceType:      string(inv.InvoiceType),
		OperationType:    inv.OperationType,
		AmountCents:      inv.AmountCents,
		Currency:         inv.Currency,
		Status:           string(inv.Status),
		ERPReference:     inv.ERPReference,
		CustomerSnapshot: inv.CustomerSnapshot,
		ItemsSnapshot:    inv.ItemsSnapshot,
		LastSyncLog:      inv.LastSyncLog,
		CreatedAt:        inv.CreatedAt,
		UpdatedAt:        inv.UpdatedAt,
	}
	if len(inv.Items) > 0 {
		resp.Items = make([]invoiceItemResponse, len(inv.Items))
		for i := range inv.Items {
			resp.Items[i] = toInvoiceItemResponse(inv.Items[i])
		}
	}
	return resp
}

func toInvoiceItemResponse(item domain.InvoiceItem) invoiceItemResponse {
	return invoiceItemResponse{
		ID:              item.ID,
		OrganizationID:  item.OrganizationID,
		InvoiceID:       item.InvoiceID,
		Description:     item.Description,
		Quantity:        item.Quantity,
		UnitPriceCents:  item.UnitPriceCents,
		TotalPriceCents: item.TotalPriceCents,
		Metadata:        item.Metadata,
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func toInvoiceLogResponse(log domain.InvoiceLog) invoiceLogResponse {
	return invoiceLogResponse{
		ID:              log.ID,
		OrganizationID:  log.OrganizationID,
		InvoiceID:       log.InvoiceID,
		EventType:       log.EventType,
		InvoiceStatus:   log.InvoiceStatus,
		ERPReference:    log.ERPReference,
		ERPErrorCode:    log.ERPErrorCode,
		ERPErrorMessage: log.ERPErrorMessage,
		HTTPStatus:      log.HTTPStatus,
		RequestURL:      log.RequestURL,
		RequestPayload:  log.RequestPayload,
		ResponseBody:    log.ResponseBody,
		AttemptNumber:   log.AttemptNumber,
		Success:         log.Success,
		Message:         log.Message,
		CreatedAt:       log.CreatedAt,
		UpdatedAt:       log.UpdatedAt,
	}
}

func toInvoiceSettingResponse(s *domain.InvoiceSetting) invoiceSettingResponse {
	return invoiceSettingResponse{
		ID:                  s.ID,
		OrganizationID:      s.OrganizationID,
		Type:                string(s.Type),
		ERPApiKey:           s.ERPApiKey,
		ERPApiSecret:        s.ERPApiSecret,
		ERPAccountID:        s.ERPAccountID,
		NfseServiceCode:     s.NfseServiceCode,
		NfseItemListService: s.NfseItemListService,
		TaxRate:             s.TaxRate,
		InvoiceSeries:       s.InvoiceSeries,
		IsoCurrency:         s.IsoCurrency,
		ExportRegime:        s.ExportRegime,
		IssueOn:             s.IssueOn,
		IsActive:            s.IsActive,
		CreatedAt:           s.CreatedAt,
		UpdatedAt:           s.UpdatedAt,
	}
}

// =============================================================================
// Invoice Handlers
// =============================================================================

// Create handles POST /v1/invoices.
func (h *InvoiceHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req createInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	items := make([]domain.CreateInvoiceItemInput, len(req.Items))
	for i, ri := range req.Items {
		totalPrice := ri.UnitPriceCents // default: quantity parsed below
		items[i] = domain.CreateInvoiceItemInput{
			Description:     ri.Description,
			Quantity:        ri.Quantity,
			UnitPriceCents:  ri.UnitPriceCents,
			TotalPriceCents: totalPrice,
		}
	}

	// Calculate total_price_cents per item and amount_cents for the invoice.
	var amountCents int32
	for i := range items {
		// Parse quantity; on error default to 1.
		var qty float64 = 1
		if v, err := parseQuantity(items[i].Quantity); err == nil {
			qty = v
		}
		items[i].TotalPriceCents = int32(qty * float64(items[i].UnitPriceCents))
		amountCents += items[i].TotalPriceCents
	}

	input := domain.CreateInvoiceInput{
		CustomerID:  req.CustomerID,
		InvoiceType: domain.InvoiceType(req.InvoiceType),
		AmountCents: amountCents,
		Items:       items,
	}

	invoice, err := h.svc.Create(c.Request.Context(), orgID, input)
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toInvoiceResponse(invoice))
}

// List handles GET /v1/invoices.
func (h *InvoiceHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	dateRange, ok := dto.ParseDateRange(c)
	if !ok {
		return
	}

	params := domain.InvoiceListParams{
		ListParams:  pag.ToListParams(),
		CreatedFrom: dateRange.From,
		CreatedTo:   dateRange.To,
	}

	if raw := c.Query("customer_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid customer_id: must be a valid UUID"))
			return
		}
		params.CustomerID = &id
	}

	if raw := c.Query("subscription_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid subscription_id: must be a valid UUID"))
			return
		}
		params.SubscriptionID = &id
	}

	if raw := c.Query("status"); raw != "" {
		s := domain.InvoiceStatus(raw)
		params.Status = &s
	}

	if raw := c.Query("invoice_type"); raw != "" {
		t := domain.InvoiceType(raw)
		params.InvoiceType = &t
	}

	invoices, total, err := h.svc.List(c.Request.Context(), orgID, params)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]invoiceResponse, len(invoices))
	for i := range invoices {
		resp[i] = toInvoiceResponse(&invoices[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[invoiceResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// GetByID handles GET /v1/invoices/:invoice_id.
func (h *InvoiceHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	invID, ok := dto.InvoiceID(c)
	if !ok {
		return
	}

	invoice, err := h.svc.GetByID(c.Request.Context(), orgID, invID)
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toInvoiceResponse(invoice))
}

// Cancel handles POST /v1/invoices/:invoice_id/cancel.
func (h *InvoiceHandler) Cancel(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	invID, ok := dto.InvoiceID(c)
	if !ok {
		return
	}

	invoice, err := h.svc.Cancel(c.Request.Context(), orgID, invID)
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toInvoiceResponse(invoice))
}

// Transition handles POST /v1/invoices/:invoice_id/transition.
func (h *InvoiceHandler) Transition(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	invID, ok := dto.InvoiceID(c)
	if !ok {
		return
	}

	var req transitionInvoiceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	invoice, err := h.svc.Transition(
		c.Request.Context(), orgID, invID,
		domain.InvoiceStatus(req.Status), req.Message,
	)
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toInvoiceResponse(invoice))
}

// ListLogs handles GET /v1/invoices/:invoice_id/logs.
func (h *InvoiceHandler) ListLogs(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	invID, ok := dto.InvoiceID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	logs, err := h.svc.ListLogs(c.Request.Context(), orgID, invID, pag.ToListParams())
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	total, err := h.svc.CountLogs(c.Request.Context(), orgID, invID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]invoiceLogResponse, len(logs))
	for i := range logs {
		resp[i] = toInvoiceLogResponse(logs[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[invoiceLogResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
		Total:  total,
	})
}

// =============================================================================
// Invoice Settings Handlers
// =============================================================================

// UpsertSetting handles PUT /v1/invoice-settings/:type.
func (h *InvoiceHandler) UpsertSetting(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	invType := c.Param("type")
	if invType != string(domain.InvTypeNational) && invType != string(domain.InvTypeInternational) {
		_ = c.Error(apierr.BadRequest("type must be one of: national, international"))
		return
	}

	var req upsertInvoiceSettingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	issueOn := req.IssueOn
	if issueOn == "" {
		issueOn = "payment_confirmed"
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	input := domain.UpsertInvoiceSettingInput{
		Type:                domain.InvoiceType(invType),
		ERPApiKey:           req.ERPApiKey,
		ERPApiSecret:        req.ERPApiSecret,
		ERPAccountID:        req.ERPAccountID,
		NfseServiceCode:     req.NfseServiceCode,
		NfseItemListService: req.NfseItemListService,
		TaxRate:             req.TaxRate,
		InvoiceSeries:       req.InvoiceSeries,
		IsoCurrency:         req.IsoCurrency,
		ExportRegime:        req.ExportRegime,
		IssueOn:             issueOn,
		IsActive:            isActive,
	}

	setting, err := h.svc.UpsertSetting(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toInvoiceSettingResponse(setting))
}

// ListSettings handles GET /v1/invoice-settings.
func (h *InvoiceHandler) ListSettings(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	settings, err := h.svc.ListSettings(c.Request.Context(), orgID)
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]invoiceSettingResponse, len(settings))
	for i := range settings {
		resp[i] = toInvoiceSettingResponse(&settings[i])
	}

	c.JSON(http.StatusOK, resp)
}

// GetSetting handles GET /v1/invoice-settings/:type.
func (h *InvoiceHandler) GetSetting(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	invType := c.Param("type")
	if invType != string(domain.InvTypeNational) && invType != string(domain.InvTypeInternational) {
		_ = c.Error(apierr.BadRequest("type must be one of: national, international"))
		return
	}

	setting, err := h.svc.GetSettingByOrgAndType(c.Request.Context(), orgID, domain.InvoiceType(invType))
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, toInvoiceSettingResponse(setting))
}

// DeleteSetting handles DELETE /v1/invoice-settings/:type.
func (h *InvoiceHandler) DeleteSetting(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	invType := c.Param("type")
	if invType != string(domain.InvTypeNational) && invType != string(domain.InvTypeInternational) {
		_ = c.Error(apierr.BadRequest("type must be one of: national, international"))
		return
	}

	// Look up the setting by type to get its ID.
	setting, err := h.svc.GetSettingByOrgAndType(c.Request.Context(), orgID, domain.InvoiceType(invType))
	if err != nil {
		mapInvoiceError(c, err)
		return
	}

	if err := h.svc.DeleteSetting(c.Request.Context(), orgID, setting.ID); err != nil {
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// =============================================================================
// Error mapping + helpers
// =============================================================================

func mapInvoiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvoiceNotFound):
		_ = c.Error(apierr.NotFound("invoice not found"))
	case errors.Is(err, domain.ErrInvoiceSettingNotFound):
		_ = c.Error(apierr.NotFound("invoice setting not found"))
	case errors.Is(err, domain.ErrCustomerNotFound):
		_ = c.Error(apierr.NotFound("customer not found"))
	case errors.Is(err, domain.ErrInvalidInvoiceTransition):
		_ = c.Error(apierr.UnprocessableEntity(err.Error()))
	case errors.Is(err, domain.ErrInvoiceAlreadyCanceled):
		_ = c.Error(apierr.UnprocessableEntity("invoice is already canceled"))
	case errors.Is(err, domain.ErrInvoiceItemsRequired):
		_ = c.Error(apierr.UnprocessableEntity("invoice must have at least one item"))
	case errors.Is(err, domain.ErrSameStateTransition):
		_ = c.Error(apierr.UnprocessableEntity("transition to the same state is not allowed"))
	case errors.Is(err, domain.ErrTerminalState):
		_ = c.Error(apierr.UnprocessableEntity("transition from terminal state is not allowed"))
	default:
		_ = c.Error(err)
	}
}

// parseQuantity parses a decimal quantity string to float64.
func parseQuantity(s string) (float64, error) {
	var v float64
	_, err := fmt.Sscanf(s, "%f", &v)
	return v, err
}
