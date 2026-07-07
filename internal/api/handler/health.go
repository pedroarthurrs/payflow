package handler

import (
	"net/http"

	"payflow/internal/api/middleware"
	"payflow/internal/database"

	"github.com/gin-gonic/gin"
)

// HealthHandler serves liveness, readiness and status probes.
type HealthHandler struct {
	healthSvc   *database.HealthChecker
	maintenance *middleware.MaintenanceProvider
}

// NewHealthHandler creates a HealthHandler backed by a HealthChecker and a MaintenanceProvider.
func NewHealthHandler(healthSvc *database.HealthChecker, maintenance *middleware.MaintenanceProvider) *HealthHandler {
	return &HealthHandler{healthSvc: healthSvc, maintenance: maintenance}
}

// RegisterRoutes registers GET /live, GET /ready and GET /status on the given router group.
func (h *HealthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/live", h.Live)
	rg.GET("/ready", h.Ready)
	rg.GET("/status", h.Status)
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

// Status returns the current maintenance flag.
func (h *HealthHandler) Status(c *gin.Context) {
	cfg := h.maintenance.Config()
	c.JSON(http.StatusOK, gin.H{"maintenance": cfg.Enabled})
}
