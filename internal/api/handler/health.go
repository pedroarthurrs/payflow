package handler

import (
	"context"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/database"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type HealthHandler struct {
	db *database.DB
}

func NewHealthHandler(db *database.DB) *HealthHandler {
	return &HealthHandler{db: db}
}

func (h *HealthHandler) Get(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
	defer cancel()

	if err := h.db.PingContext(ctx); err != nil {
		_ = c.Error(apierr.ServiceUnavailable("database unavailable: " + err.Error()))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"services": gin.H{
			"database": "ok",
		},
	})
}
