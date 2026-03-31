package handler

import (
	"net/http"

	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

// HealthHandler serves liveness and readiness probes.
type HealthHandler struct {
	healthSvc *service.HealthService
}

// NewHealthHandler creates a HealthHandler backed by a HealthService.
func NewHealthHandler(healthSvc *service.HealthService) *HealthHandler {
	return &HealthHandler{healthSvc: healthSvc}
}

// RegisterRoutes registers GET /live and GET /ready on the given router group.
func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/live", h.Live)
	rg.GET("/ready", h.Ready)
}

// Live returns 200 {"status": "alive"} unconditionally.
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "alive"})
}

// Ready returns 200 {"status": "ready"} or 503 {"status": "unavailable"}
// based on the cached database health state.
func (h *HealthHandler) Ready(c *gin.Context) {
	if h.healthSvc.DatabaseOK() {
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
		return
	}
	c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
}
