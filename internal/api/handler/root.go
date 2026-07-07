package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// RootHandler handles the root endpoint.
type RootHandler struct{}

// NewRootHandler creates a RootHandler.
func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

// RegisterRoutes registers the root handler routes.
func (h *RootHandler) RegisterRoutes(rg *gin.RouterGroup) {
	rg.GET("/", h.Get)
}

// Get returns basic API information.
func (h *RootHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name": "payflow",
	})
}
