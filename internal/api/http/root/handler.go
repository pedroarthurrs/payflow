package root

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles the root endpoint.
type Handler struct{}

// NewHandler creates a root Handler.
func NewHandler() *Handler {
	return &Handler{}
}

// Get returns basic API information.
func (h *Handler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name": "i9pay",
	})

}
