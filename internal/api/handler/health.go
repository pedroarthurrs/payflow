package handler

import (
	"context"
	"net/http"
	"time"

	"i9pay/internal/api/err"
	"i9pay/internal/database"

	"github.com/gin-gonic/gin"
)

// HealthHandler handles health check endpoints.
type HealthHandler struct {
	db *database.DB
}

// NewHealthHandler creates a HealthHandler.
func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

// RegisterRoutes registers the health handler routes.
func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/health", h.Get)
}

// Get returns the health status of the service and its dependencies.
func (h *HealthHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if e := h.db.PingContext(ctx); e != nil {
		_ = c.Error(err.ServiceUnavailable("database unavailable: " + e.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"services": gin.H{
			"database": "ok",
		},
	})
}
