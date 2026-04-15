package handler

import (
	"errors"
	"net/http"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	auth.POST("/select-organization", h.SelectOrganization)
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	APIKey   string `json:"api_key"`
}

// orgRefResponse is a single item in the multi-org login response.
type orgRefResponse struct {
	OrganizationID string      `json:"organization_id"`
	Name           string      `json:"name"`
	Role           domain.Role `json:"role"`
}

// multiOrgLoginResponse is returned when the user belongs to multiple organizations.
type multiOrgLoginResponse struct {
	SessionToken  string           `json:"session_token"`
	Organizations []orgRefResponse `json:"organizations"`
}

// Login authenticates via dynamic mode detection and returns either a token pair
// (single-org) or a session token + org list (multi-org).
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}

	result, e := h.authSvc.Login(c.Request.Context(), domain.LoginRequest{
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

	// Single-org: return the full token pair directly.
	if result.TokenPair != nil {
		c.JSON(http.StatusOK, result.TokenPair)
		return
	}

	// Multi-org: return session token + org list.
	orgs := make([]orgRefResponse, len(result.Organizations))
	for i, m := range result.Organizations {
		orgs[i] = orgRefResponse{
			OrganizationID: m.OrganizationID.String(),
			Name:           m.Name,
			Role:           m.Role,
		}
	}
	c.JSON(http.StatusOK, multiOrgLoginResponse{
		SessionToken:  *result.SessionToken,
		Organizations: orgs,
	})
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

type selectOrgRequest struct {
	SessionToken   string `json:"session_token"    binding:"required"`
	OrganizationID string `json:"organization_id"  binding:"required"`
}

// SelectOrganization validates an org_selection session token and issues a full
// access/refresh token pair for the chosen organization.
func (h *AuthHandler) SelectOrganization(c *gin.Context) {
	var req selectOrgRequest
	if e := c.ShouldBindJSON(&req); e != nil {
		_ = c.Error(apierr.BadRequest(e.Error()))
		return
	}

	orgID, err := uuid.Parse(req.OrganizationID)
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid organization_id"))
		return
	}

	pair, e := h.authSvc.SelectOrganization(c.Request.Context(), req.SessionToken, orgID)
	if e != nil {
		if errors.Is(e, domain.ErrInvalidSessionToken) {
			_ = c.Error(apierr.Unauthorized("invalid or expired session token"))
			return
		}
		if errors.Is(e, domain.ErrForbidden) {
			_ = c.Error(apierr.Forbidden("not a member of that organization"))
			return
		}
		_ = c.Error(apierr.Unauthorized("invalid credentials"))
		return
	}

	c.JSON(http.StatusOK, pair)
}
