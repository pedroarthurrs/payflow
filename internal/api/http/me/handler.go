package me

import (
	"net/http"

	"i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/token"

	"github.com/gin-gonic/gin"
)

// Handler handles the authenticated user profile endpoints.
type Handler struct{}

// NewHandler creates a me Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Get returns basic profile information extracted from the access token claims.
func (h *Handler) Get(c *gin.Context) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(err.Internal("could not parse claims"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":  claims.TenantID,
		"issued_at":  claims.IssuedAt,
		"expires_at": claims.ExpiresAt,
	})
}
