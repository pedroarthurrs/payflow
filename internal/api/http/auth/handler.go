package auth

import (
	"i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

// Handler handles authentication-related HTTP endpoints.
type Handler struct {
	authSvc *service.AuthService
}

// NewHandler creates an auth Handler.
func NewHandler(authSvc *service.AuthService) *Handler {
	return &Handler{authSvc: authSvc}
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Login authenticates a tenant and returns a fresh access/refresh token pair.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(err.BadRequest(e.Error()))
		return
	}

	pair, e := h.authSvc.Login(domain.Credentials{
		Email:    req.Email,
		Password: req.Password,
	})
	if e != nil {
		_ = c.Error(err.Unauthorized("invalid credentials"))
		return
	}

	c.JSON(http.StatusOK, pair)
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh validates the supplied refresh token and returns a new token pair.
func (h *Handler) Refresh(c *gin.Context) {
	var req refreshRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(err.BadRequest(e.Error()))
		return
	}

	pair, e := h.authSvc.RefreshToken(req.RefreshToken)
	if e != nil {
		_ = c.Error(err.Unauthorized("invalid or expired refresh token"))
		return
	}

	c.JSON(http.StatusOK, pair)
}
