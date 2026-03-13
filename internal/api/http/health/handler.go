package health

import (
	"context"
	"i9pay/internal/api/err"
	"i9pay/internal/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Handler handles health check endpoints.
type Handler struct {
	db *database.DB
}

// NewHandler creates a health Handler.
func NewHandler(db *database.DB) *Handler {
	return &Handler{db: db}
}

// Get returns the health status of the service and its dependencies.
func (h *Handler) Get(c *gin.Context) {
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
