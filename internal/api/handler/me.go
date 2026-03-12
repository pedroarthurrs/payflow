package handler

import (
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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not parse claims"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"name":       claims.Name,
		"password":   claims.Password,
		"issued_at":  claims.IssuedAt,
		"expires_at": claims.ExpiresAt,
	})
}
