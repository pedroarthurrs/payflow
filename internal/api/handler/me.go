package handler

import (
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

type MeHandler struct{}

func NewMeHandler() *MeHandler {
	return &MeHandler{}
}

func (h *MeHandler) Get(c *gin.Context) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*middleware.Claims)
	if !ok {
		_ = c.Error(apierr.Internal("could not parse claims"))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"email":      claims.Email,
		"password":   claims.Password,
		"issued_at":  claims.IssuedAt,
		"expires_at": claims.ExpiresAt,
	})
}
