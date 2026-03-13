package handler

import (
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	i9token "i9pay/internal/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MeHandler struct{}

func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

// Get returns basic profile information extracted from the authenticated
// tenant's access token claims.
func (h *MeHandler) Get(c *gin.Context) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*i9token.Claims)
	if !ok {
		_ = c.Error(apierr.Internal("could not parse claims"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id":  claims.TenantID,
		"issued_at":  claims.IssuedAt,
		"expires_at": claims.ExpiresAt,
	})
}
