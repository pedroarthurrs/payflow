package middleware

import (
	"net/http"
	"sync/atomic"

	"github.com/gin-gonic/gin"
)

// MaintenanceConfig holds the maintenance mode settings.
type MaintenanceConfig struct {
	Enabled     bool   `json:"enabled"`
	Message     string `json:"message,omitempty"`
	AvailableAt string `json:"available_at,omitempty"`
}

// MaintenanceProvider holds the current maintenance state thread-safely
// and exposes a Gin middleware. Use NewMaintenanceProvider to create one.
type MaintenanceProvider struct {
	state atomic.Value
}

// maintenanceEnabled controls whether maintenance mode is active.
// Set to true to enable maintenance mode manually.
const maintenanceEnabled = false

// NewMaintenanceProvider returns a MaintenanceProvider initialized with maintenanceEnabled.
func NewMaintenanceProvider() *MaintenanceProvider {
	p := &MaintenanceProvider{}
	p.state.Store(MaintenanceConfig{Enabled: maintenanceEnabled})
	return p
}

// Set updates the maintenance configuration.
func (p *MaintenanceProvider) Set(cfg MaintenanceConfig) {
	p.state.Store(cfg)
}

// Config returns the current maintenance configuration.
func (p *MaintenanceProvider) Config() MaintenanceConfig {
	if v := p.state.Load(); v != nil {
		return v.(MaintenanceConfig)
	}
	return MaintenanceConfig{}
}

// Maintenance returns a Gin middleware that blocks requests when maintenance is enabled.
// The health/status endpoints are always allowed through.
func Maintenance(p *MaintenanceProvider) gin.HandlerFunc {
	return func(c *gin.Context) {
		cfg := p.Config()
		if !cfg.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if path == "/v1/live" || path == "/v1/ready" || path == "/v1/status" {
			c.Next()
			return
		}

		c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{
			"code":         "maintenance",
			"message":      cfg.Message,
			"available_at": cfg.AvailableAt,
		})
	}
}
