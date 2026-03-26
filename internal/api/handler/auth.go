package handler

import (
	"errors"
	"net/http"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	authSvc *service.AuthService
	loginRL gin.HandlerFunc
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authSvc *service.AuthService, loginRL gin.HandlerFunc) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, loginRL: loginRL}
}

// RegisterRoutes registers the auth handler routes.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/login", h.loginRL, h.Login)
	auth.POST("/refresh", h.Refresh)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
}

// Login authenticates via dynamic mode detection and returns a token pair.
func (h *AuthHandler) Login(c *gin.Context) {
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
func (h *AuthHandler) Refresh(c *gin.Context) {
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
