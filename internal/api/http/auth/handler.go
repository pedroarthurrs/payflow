package auth

import (
	"errors"
	"net/http"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"

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
	Email    string `json:"email"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
}

// Login authenticates via dynamic mode detection and returns a token pair.
func (h *Handler) Login(c *gin.Context) {
	var req loginRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}

	pair, e := h.authSvc.Login(c.Request.Context(), domain.LoginRequest{
		Email:    req.Email,
		Password: req.Password,
		APIKey:   req.APIKey,
	})
	if e != nil {
		if errors.Is(e, domain.ErrBadRequest) {
			_ = c.Error(apierr.BadRequest(e.Error()))
			return
		}
		_ = c.Error(apierr.Unauthorized("invalid credentials"))
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
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}

	pair, e := h.authSvc.RefreshToken(req.RefreshToken)
	if e != nil {
		_ = c.Error(apierr.Unauthorized("invalid or expired refresh token"))
		return
	}

	c.JSON(http.StatusOK, pair)
}
