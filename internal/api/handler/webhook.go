package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"payflow/internal/api/dto"
	apierr "payflow/internal/api/err"
	"payflow/internal/api/middleware"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// WebhookHandler handles webhook-related HTTP endpoints.
type WebhookHandler struct {
	svc *service.WebhookService
}

// NewWebhookHandler creates a WebhookHandler.
func NewWebhookHandler(svc *service.WebhookService) *WebhookHandler {
	return &WebhookHandler{svc: svc}
}

// RegisterRoutes registers the webhook handler routes.
// IMPORTANT: /events, /logs and /logs/:log_id are registered BEFORE /:webhook_id
// to avoid route conflicts with Gin's parameter matching.
func (h *WebhookHandler) RegisterRoutes(rg *gin.RouterGroup) {
	webhooks := rg.Group("/webhooks")

	// Endpoints CRUD (admin)
	webhooks.POST("", middleware.RequireRole(domain.RoleAdmin), h.Create)
	webhooks.GET("", middleware.RequireRole(domain.RoleAdmin), h.List)
	webhooks.GET("/events", middleware.RequireRole(domain.RoleAdmin), h.ListEvents)

	// Logs (admin) — registered BEFORE /:webhook_id to avoid route conflict
	webhooks.GET("/logs", middleware.RequireRole(domain.RoleAdmin), h.ListLogs)
	webhooks.GET("/logs/:log_id", middleware.RequireRole(domain.RoleAdmin), h.GetLogByID)

	// Endpoints by ID
	webhooks.GET("/:webhook_id", middleware.RequireRole(domain.RoleAdmin), h.GetByID)
	webhooks.PUT("/:webhook_id", middleware.RequireRole(domain.RoleAdmin), h.Update)
	webhooks.DELETE("/:webhook_id", middleware.RequireRole(domain.RoleOwner), h.Delete)
}

// --- Request / Response DTOs ---

type createWebhookRequest struct {
	URL           string   `json:"url"           binding:"required,url,startswith=https://"`
	Description   *string  `json:"description"   binding:"omitempty,max=255"`
	EnabledEvents []string `json:"enabled_events" binding:"required,min=1"`
	Environment   *string  `json:"environment"   binding:"omitempty,oneof=test production"`
}

type updateWebhookRequest struct {
	URL           *string  `json:"url"            binding:"omitempty,url,startswith=https://"`
	Description   *string  `json:"description"    binding:"omitempty,max=255"`
	EnabledEvents []string `json:"enabled_events" binding:"omitempty,min=1"`
	IsActive      *bool    `json:"is_active"`
}

type webhookEndpointResponse struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	URL            string    `json:"url"`
	Description    *string   `json:"description"`
	EnabledEvents  []string  `json:"enabled_events"`
	Environment    string    `json:"environment"`
	IsActive       bool      `json:"is_active"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type webhookLogResponse struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	EndpointID     uuid.UUID       `json:"endpoint_id"`
	EventType      string          `json:"event_type"`
	Payload        json.RawMessage `json:"payload"`
	ResponseBody   *string         `json:"response_body"`
	ResponseStatus *int32          `json:"response_status"`
	DurationMs     *int32          `json:"duration_ms"`
	AttemptNumber  int32           `json:"attempt_number"`
	Success        bool            `json:"success"`
	CreatedAt      time.Time       `json:"created_at"`
}

type listWebhookLogsResponse struct {
	Data       []webhookLogResponse `json:"data"`
	NextCursor *string              `json:"next_cursor"`
	Limit      int                  `json:"limit"`
}

func toWebhookEndpointResponse(e *domain.WebhookEndpoint) webhookEndpointResponse {
	return webhookEndpointResponse{
		ID:             e.ID,
		OrganizationID: e.OrganizationID,
		URL:            e.URL,
		Description:    e.Description,
		EnabledEvents:  e.EnabledEvents,
		Environment:    e.Environment,
		IsActive:       e.IsActive,
		CreatedAt:      e.CreatedAt,
		UpdatedAt:      e.UpdatedAt,
	}
}

func toWebhookLogResponse(l domain.WebhookLog) webhookLogResponse {
	return webhookLogResponse{
		ID:             l.ID,
		OrganizationID: l.OrganizationID,
		EndpointID:     l.EndpointID,
		EventType:      l.EventType,
		Payload:        l.Payload,
		ResponseBody:   l.ResponseBody,
		ResponseStatus: l.ResponseStatus,
		DurationMs:     l.DurationMs,
		AttemptNumber:  l.AttemptNumber,
		Success:        l.Success,
		CreatedAt:      l.CreatedAt,
	}
}

// --- Helpers ---

// validateEventTypes checks that all provided event types are valid.
// Returns a list of invalid types, or nil if all are valid.
func validateEventTypes(events []string) []string {
	var invalid []string
	for _, et := range events {
		if !domain.IsValidEventType(et) {
			invalid = append(invalid, et)
		}
	}
	return invalid
}

// --- Handlers ---

// Create handles POST /v1/webhooks.
func (h *WebhookHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req createWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	if invalid := validateEventTypes(req.EnabledEvents); len(invalid) > 0 {
		_ = c.Error(apierr.BadRequest(fmt.Sprintf("invalid event types: [%s]", strings.Join(invalid, ", "))))
		return
	}

	env := "test"
	if req.Environment != nil {
		env = *req.Environment
	}

	endpoint, err := h.svc.CreateEndpoint(c.Request.Context(), orgID, domain.CreateWebhookEndpointInput{
		URL:           req.URL,
		Description:   req.Description,
		EnabledEvents: req.EnabledEvents,
		Environment:   env,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, toWebhookEndpointResponse(endpoint))
}

// List handles GET /v1/webhooks.
func (h *WebhookHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}

	endpoints, err := h.svc.ListEndpoints(c.Request.Context(), orgID, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]webhookEndpointResponse, len(endpoints))
	for i := range endpoints {
		resp[i] = toWebhookEndpointResponse(&endpoints[i])
	}

	c.JSON(http.StatusOK, dto.PagedResponse[webhookEndpointResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
	})
}

// GetByID handles GET /v1/webhooks/:webhook_id.
func (h *WebhookHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	whID, ok := dto.WebhookID(c)
	if !ok {
		return
	}

	endpoint, err := h.svc.GetEndpointByID(c.Request.Context(), orgID, whID)
	if err != nil {
		if errors.Is(err, domain.ErrWebhookEndpointNotFound) {
			_ = c.Error(apierr.NotFound("webhook endpoint not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toWebhookEndpointResponse(endpoint))
}

// Update handles PUT /v1/webhooks/:webhook_id.
func (h *WebhookHandler) Update(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	whID, ok := dto.WebhookID(c)
	if !ok {
		return
	}

	var req updateWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}

	if len(req.EnabledEvents) > 0 {
		if invalid := validateEventTypes(req.EnabledEvents); len(invalid) > 0 {
			_ = c.Error(apierr.BadRequest(fmt.Sprintf("invalid event types: [%s]", strings.Join(invalid, ", "))))
			return
		}
	}

	endpoint, err := h.svc.UpdateEndpoint(c.Request.Context(), orgID, whID, domain.UpdateWebhookEndpointInput{
		URL:           req.URL,
		Description:   req.Description,
		EnabledEvents: req.EnabledEvents,
		IsActive:      req.IsActive,
	})
	if err != nil {
		if errors.Is(err, domain.ErrWebhookEndpointNotFound) {
			_ = c.Error(apierr.NotFound("webhook endpoint not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toWebhookEndpointResponse(endpoint))
}

// Delete handles DELETE /v1/webhooks/:webhook_id.
func (h *WebhookHandler) Delete(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	whID, ok := dto.WebhookID(c)
	if !ok {
		return
	}

	err := h.svc.DeleteEndpoint(c.Request.Context(), orgID, whID)
	if err != nil {
		if errors.Is(err, domain.ErrWebhookEndpointNotFound) {
			_ = c.Error(apierr.NotFound("webhook endpoint not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}

// ListLogs handles GET /v1/webhooks/logs.
func (h *WebhookHandler) ListLogs(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	pag, ok := dto.ParsePagination(c)
	if !ok {
		return
	}

	dates, ok := dto.ParseDateRange(c)
	if !ok {
		return
	}

	// Parse endpoint_id filter
	var endpointID *uuid.UUID
	if raw := c.Query("endpoint_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("endpoint_id must be a valid UUID"))
			return
		}
		endpointID = &id
	}

	// Parse success filter
	var success *bool
	if raw := c.Query("success"); raw != "" {
		v, err := strconv.ParseBool(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("success must be a boolean"))
			return
		}
		success = &v
	}

	eventType := dto.ParseString(c, "event_type")

	input := domain.ListWebhookLogsInput{
		Limit:      pag.Limit,
		Cursor:     pag.Cursor,
		From:       dates.From,
		To:         dates.To,
		EndpointID: endpointID,
		EventType:  eventType,
		Success:    success,
	}

	logs, nextCursor, err := h.svc.ListLogs(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	data := make([]webhookLogResponse, len(logs))
	for i := range logs {
		data[i] = toWebhookLogResponse(logs[i])
	}

	c.JSON(http.StatusOK, listWebhookLogsResponse{
		Data:       data,
		NextCursor: nextCursor,
		Limit:      pag.Limit,
	})
}

// GetLogByID handles GET /v1/webhooks/logs/:log_id.
func (h *WebhookHandler) GetLogByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	logID, ok := dto.LogID(c)
	if !ok {
		return
	}

	record, err := h.svc.GetLogByID(c.Request.Context(), orgID, logID)
	if err != nil {
		if errors.Is(err, domain.ErrWebhookLogNotFound) {
			_ = c.Error(apierr.NotFound("webhook log not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toWebhookLogResponse(*record))
}

// ListEvents handles GET /v1/webhooks/events.
func (h *WebhookHandler) ListEvents(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"events": h.svc.ListEventTypes()})
}
