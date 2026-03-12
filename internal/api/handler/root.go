package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type RootHandler struct{}

func NewRootHandler() *RootHandler {
	return &RootHandler{}
}

func (h *RootHandler) Get(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"name": "i9pay",
	})
}
