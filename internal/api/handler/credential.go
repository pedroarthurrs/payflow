package handler

import (
	"errors"
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

// CredentialHandler handles credential-related HTTP endpoints.
type CredentialHandler struct {
	credentialSvc *service.CredentialService
}

// NewCredentialHandler creates a CredentialHandler.
func NewCredentialHandler(credentialSvc *service.CredentialService) *CredentialHandler {
	return &CredentialHandler{credentialSvc: credentialSvc}
}

// RegisterRoutes registers the credential handler routes.
func (h *CredentialHandler) RegisterRoutes(rg *gin.RouterGroup) {
	creds := rg.Group("/credentials")
	creds.POST("", middleware.RequireRole(domain.RoleAdmin), h.Create)
	creds.GET("", middleware.RequireRole(domain.RoleAdmin), h.List)
	creds.GET("/:credential_id", middleware.RequireRole(domain.RoleAdmin), h.GetByID)
	creds.DELETE("/:credential_id", middleware.RequireRole(domain.RoleOwner), h.Revoke)
	creds.POST("/:credential_id/rotate", middleware.RequireRole(domain.RoleOwner), h.Rotate)
}

// createCredentialRequest is the DTO for creating a credential.
type createCredentialRequest struct {
	Name        string     `json:"name"        binding:"required,min=1,max=50"`
	Environment string     `json:"environment" binding:"required,oneof=test production"`
	ExpiresAt   *time.Time `json:"expires_at"`
}

// credentialResponse is the DTO for credential responses (no raw_key, no key_hash).
type credentialResponse struct {
	ID          uuid.UUID  `json:"id"`
	Name        string     `json:"name"`
	KeyPrefix   string     `json:"key_prefix"`
	Environment string     `json:"environment"`
	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

// createCredentialResponse is the DTO for create/rotate responses (includes raw_key and warning).
type createCredentialResponse struct {
	credentialResponse
	RawKey  string `json:"raw_key"`
	Warning string `json:"warning"`
}

const credentialWarning = "Guarde esta chave agora. Ela não poderá ser recuperada novamente."

func toCredentialResponse(r *domain.CredentialRecord) credentialResponse {
	return credentialResponse{
		ID:          r.ID,
		Name:        r.Name,
		KeyPrefix:   r.KeyPrefix,
		Environment: r.Environment,
		LastUsedAt:  r.LastUsedAt,
		ExpiresAt:   r.ExpiresAt,
		CreatedAt:   r.CreatedAt,
	}
}

// Create handles POST /v1/credentials.
func (h *CredentialHandler) Create(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	var req createCredentialRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(err)
		return
	}
	record, rawKey, err := h.credentialSvc.Create(c.Request.Context(), orgID, domain.CreateCredentialInput{
		Name:        req.Name,
		Environment: req.Environment,
		ExpiresAt:   req.ExpiresAt,
	})
	if err != nil {
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, createCredentialResponse{
		credentialResponse: toCredentialResponse(record),
		RawKey:             rawKey,
		Warning:            credentialWarning,
	})
}

// List handles GET /v1/credentials.
func (h *CredentialHandler) List(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	pag, ok := dto.ParseOffsetPagination(c)
	if !ok {
		return
	}
	creds, err := h.credentialSvc.List(c.Request.Context(), orgID, pag.ToListParams())
	if err != nil {
		_ = c.Error(err)
		return
	}
	resp := make([]credentialResponse, len(creds))
	for i := range creds {
		resp[i] = toCredentialResponse(&creds[i])
	}
	c.JSON(http.StatusOK, dto.PagedResponse[credentialResponse]{
		Data:   resp,
		Limit:  pag.Limit,
		Offset: pag.Offset,
	})
}

// GetByID handles GET /v1/credentials/:credential_id.
func (h *CredentialHandler) GetByID(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	credID, ok := dto.CredentialID(c)
	if !ok {
		return
	}
	record, err := h.credentialSvc.GetByID(c.Request.Context(), orgID, credID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			_ = c.Error(apierr.NotFound("credential not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusOK, toCredentialResponse(record))
}

// Revoke handles DELETE /v1/credentials/:credential_id.
func (h *CredentialHandler) Revoke(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	credID, ok := dto.CredentialID(c)
	if !ok {
		return
	}
	err := h.credentialSvc.Revoke(c.Request.Context(), orgID, credID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			_ = c.Error(apierr.NotFound("credential not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.Status(http.StatusNoContent)
}

// Rotate handles POST /v1/credentials/:credential_id/rotate.
func (h *CredentialHandler) Rotate(c *gin.Context) {
	orgID, ok := dto.OrganizationID(c)
	if !ok {
		return
	}
	credID, ok := dto.CredentialID(c)
	if !ok {
		return
	}
	record, rawKey, err := h.credentialSvc.Rotate(c.Request.Context(), orgID, credID)
	if err != nil {
		if errors.Is(err, domain.ErrCredentialNotFound) {
			_ = c.Error(apierr.NotFound("credential not found"))
			return
		}
		_ = c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, createCredentialResponse{
		credentialResponse: toCredentialResponse(record),
		RawKey:             rawKey,
		Warning:            credentialWarning,
	})
}
