package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

// CredentialLogHandler handles credential log read endpoints.
type CredentialLogHandler struct {
	svc *service.CredentialLogService
}

// NewCredentialLogHandler creates a CredentialLogHandler.
func NewCredentialLogHandler(svc *service.CredentialLogService) *CredentialLogHandler {
	return &CredentialLogHandler{svc: svc}
}

// RegisterRoutes registers the credential log routes under a separate group
// to avoid Gin parameter conflicts with existing credential routes.
func (h *CredentialLogHandler) RegisterRoutes(rg *gin.RouterGroup) {
	logs := rg.Group("/credentials/logs")
	logs.Use(middleware.RequireRole(domain.RoleAdmin))
	logs.GET("", h.List)
	logs.GET("/:log_id", h.GetByID)
}

// credentialLogResponse is the DTO for a single credential log entry.
type credentialLogResponse struct {
	ID             string          `json:"id"`
	OrganizationID string          `json:"organization_id"`
	CredentialID   *string         `json:"credential_id"`
	Method         *string         `json:"method"`
	Path           *string         `json:"path"`
	RequestBody    json.RawMessage `json:"request_body"`
	ResponseBody   json.RawMessage `json:"response_body"`
	StatusCode     *int32          `json:"status_code"`
	LatencyMs      *int32          `json:"latency_ms"`
	IPAddress      *string         `json:"ip_address"`
	UserAgent      *string         `json:"user_agent"`
	CreatedAt      time.Time       `json:"created_at"`
}

// listCredentialLogsResponse is the paginated list response.
type listCredentialLogsResponse struct {
	Data       []credentialLogResponse `json:"data"`
	NextCursor *string                 `json:"next_cursor"`
	Limit      int                     `json:"limit"`
}

func toCredentialLogResponse(r domain.CredentialLogRecord) credentialLogResponse {
	resp := credentialLogResponse{
		ID:             r.ID.String(),
		OrganizationID: r.OrganizationID.String(),
		Method:         r.Method,
		Path:           r.Path,
		RequestBody:    r.RequestBody,
		ResponseBody:   r.ResponseBody,
		StatusCode:     r.StatusCode,
		LatencyMs:      r.LatencyMs,
		IPAddress:      r.IPAddress,
		UserAgent:      r.UserAgent,
		CreatedAt:      r.CreatedAt,
	}
	if r.CredentialID != nil {
		s := r.CredentialID.String()
		resp.CredentialID = &s
	}
	return resp
}

const (
	defaultLimit = 20
	maxLimit     = 100
)

// List handles GET /v1/credentials/logs.
func (h *CredentialLogHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	// Parse limit.
	limit := defaultLimit
	if raw := c.Query("limit"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil || v < 1 || v > maxLimit {
			_ = c.Error(apierr.BadRequest("limit must be an integer between 1 and 100"))
			return
		}
		limit = v
	}

	// Parse cursor.
	var cursorToken *domain.CursorToken
	if raw := c.Query("cursor"); raw != "" {
		ct, err := service.DecodeCursor(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("invalid cursor"))
			return
		}
		cursorToken = ct
	}

	// Parse from/to timestamps.
	var from, to *time.Time
	if raw := c.Query("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("from must be a valid ISO 8601 timestamp"))
			return
		}
		from = &t
	}
	if raw := c.Query("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("to must be a valid ISO 8601 timestamp"))
			return
		}
		to = &t
	}
	if from != nil && to != nil && from.After(*to) {
		_ = c.Error(apierr.BadRequest("from must not be after to"))
		return
	}

	// Parse status_code.
	var statusCode *int32
	if raw := c.Query("status_code"); raw != "" {
		v, err := strconv.Atoi(raw)
		if err != nil {
			_ = c.Error(apierr.BadRequest("status_code must be a valid integer"))
			return
		}
		sc := int32(v)
		statusCode = &sc
	}

	// Parse method and path.
	var method, pathPrefix *string
	if raw := c.Query("method"); raw != "" {
		method = &raw
	}
	if raw := c.Query("path"); raw != "" {
		pathPrefix = &raw
	}

	input := domain.ListCredentialLogsInput{
		Limit:      limit,
		Cursor:     cursorToken,
		From:       from,
		To:         to,
		StatusCode: statusCode,
		Method:     method,
		PathPrefix: pathPrefix,
	}

	records, nextCursor, err := h.svc.List(c.Request.Context(), orgID, input)
	if err != nil {
		_ = c.Error(err)
		return
	}

	data := make([]credentialLogResponse, len(records))
	for i := range records {
		data[i] = toCredentialLogResponse(records[i])
	}

	c.JSON(http.StatusOK, listCredentialLogsResponse{
		Data:       data,
		NextCursor: nextCursor,
		Limit:      limit,
	})
}

// GetByID handles GET /v1/credentials/logs/:log_id.
func (h *CredentialLogHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	logID, ok := dto.LogID(c)
	if !ok {
		return
	}

	record, err := h.svc.GetByID(c.Request.Context(), orgID, logID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialLogNotFound) {
			_ = c.Error(apierr.NotFound("credential log not found"))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toCredentialLogResponse(*record))
}
