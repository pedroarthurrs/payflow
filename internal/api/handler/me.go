package handler

import (
	"net/http"

	"payflow/internal/api/err"
	"payflow/internal/api/middleware"
	"payflow/internal/auth/token"
	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MeHandler handles the authenticated user profile endpoints.
type MeHandler struct {
	permSvc *service.PermissionService
}

// NewMeHandler creates a MeHandler.
func NewMeHandler(permSvc *service.PermissionService) *MeHandler {
	return &MeHandler{permSvc: permSvc}
}

// RegisterRoutes registers the me handler routes.
func (h *MeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/me", h.Get)
}

type meResponse struct {
	OrganizationID string              `json:"organization_id"`
	UserID         string              `json:"user_id,omitempty"`
	Role           domain.Role         `json:"role,omitempty"`
	Permissions    []domain.Permission `json:"permissions,omitempty"`
	IssuedAt       interface{}         `json:"issued_at"`
	ExpiresAt      interface{}         `json:"expires_at"`
}

// Get returns profile information extracted from the access token claims,
// including the effective permission set for the authenticated user.
func (h *MeHandler) Get(c *gin.Context) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(err.Internal("could not parse claims"))
		return
	}

	resp := meResponse{
		OrganizationID: claims.OrganizationID,
		UserID:         claims.UserID,
		Role:           claims.Role,
		IssuedAt:       claims.IssuedAt,
		ExpiresAt:      claims.ExpiresAt,
	}

	// Populate effective permissions for user tokens.
	if h.permSvc != nil && claims.UserID != "" {
		orgID, errOrg := uuid.Parse(claims.OrganizationID)
		userID, errUser := uuid.Parse(claims.UserID)
		if errOrg == nil && errUser == nil {
			if perms, e := h.permSvc.EffectivePermissions(c.Request.Context(), orgID, userID); e == nil {
				resp.Permissions = perms
			}
		}
	}

	c.JSON(http.StatusOK, resp)
}
