package handler

import (
	"net/http"

	"i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"

	"github.com/gin-gonic/gin"
)

// MeHandler handles the authenticated user profile endpoints.
type MeHandler struct{}

// NewMeHandler creates a MeHandler.
func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

// RegisterRoutes registers the me handler routes.
func (h *MeHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/me", h.Get)
}

// Get returns basic profile information extracted from the access token claims.
func (h *MeHandler) Get(c *gin.Context) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(err.Internal("could not parse claims"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"organization_id": claims.OrganizationID,
		"issued_at":       claims.IssuedAt,
		"expires_at":      claims.ExpiresAt,
	})
}
