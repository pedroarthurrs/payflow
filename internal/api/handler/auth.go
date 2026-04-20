package handler

import (
	"errors"
	"net/http"
	"time"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	accessTokenCookieName  = "i9pay_access_token"
	refreshTokenCookieName = "i9pay_refresh_token"
)

// AuthHandler handles authentication-related HTTP endpoints.
type AuthHandler struct {
	authSvc    *service.AuthService
	loginRL    gin.HandlerFunc
	production bool // controls Secure + SameSite flags on auth cookies
}

// NewAuthHandler creates an AuthHandler.
// production should be true when running in a TLS-enabled environment so that
// the HttpOnly cookies receive the Secure flag.
func NewAuthHandler(authSvc *service.AuthService, loginRL gin.HandlerFunc, production bool) *AuthHandler {
	return &AuthHandler{authSvc: authSvc, loginRL: loginRL, production: production}
}

// setAuthCookies writes the access and refresh tokens as HttpOnly cookies and
// refreshes the CSRF token cookie so the browser's JS layer can echo it back.
// Returns an error if the CSRF token cannot be generated.
func (h *AuthHandler) setAuthCookies(c *gin.Context, pair *token.Pair) error {
	secure := h.production

	accessAge := int(time.Until(pair.AccessTokenExpiresAt).Seconds())
	if accessAge <= 0 {
		accessAge = 900 // fallback: 15 min
	}
	c.SetCookie(accessTokenCookieName, pair.AccessToken, accessAge, "/", "", secure, true /* HttpOnly */)

	refreshAge := int(time.Until(pair.RefreshTokenExpiresAt).Seconds())
	if refreshAge <= 0 {
		refreshAge = 7 * 24 * 3600 // fallback: 7 days
	}
	// Scope refresh cookie to auth path — the browser only sends it on /v1/auth/*.
	c.SetCookie(refreshTokenCookieName, pair.RefreshToken, refreshAge, "/v1/auth", "", secure, true /* HttpOnly */)

	// Refresh CSRF token so it stays in sync with the new session.
	if _, err := middleware.SetCSRFCookie(c, secure); err != nil {
		return err
	}
	return nil
}

// RegisterRoutes registers the auth handler routes.
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup) {
	auth := rg.Group("/auth")
	auth.POST("/login", h.loginRL, h.Login)
	auth.POST("/refresh", h.Refresh)
	auth.POST("/select-organization", h.SelectOrganization)
	auth.POST("/logout", h.Logout)
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

	// Single-org: set auth cookies and return the full token pair in the body
	// (body allows non-browser clients; HttpOnly cookies serve browser clients).
	if result.TokenPair != nil {
		if err := h.setAuthCookies(c, result.TokenPair); err != nil {
			_ = c.Error(apierr.Internal("failed to set auth cookies"))
			return
		}
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
	RefreshToken string `json:"refresh_token"`
}

// Refresh validates the supplied refresh token and returns a new token pair.
// The token may be provided either in the JSON body ({"refresh_token": "..."})
// or via the HttpOnly "i9pay_refresh_token" cookie (browser clients).
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	_ = c.ShouldBindJSON(&req) // non-fatal if body is absent or empty

	// Fall back to HttpOnly cookie when the body does not contain the token.
	if req.RefreshToken == "" {
		cookie, err := c.Cookie(refreshTokenCookieName)
		if err != nil || cookie == "" {
			_ = c.Error(apierr.BadRequest("refresh_token required (body or cookie)"))
			return
		}
		req.RefreshToken = cookie
	}

	pair, e := h.authSvc.RefreshToken(req.RefreshToken)
	if e != nil {
		_ = c.Error(apierr.Unauthorized("invalid or expired refresh token"))
		return
	}

	if err := h.setAuthCookies(c, pair); err != nil {
		_ = c.Error(apierr.Internal("failed to set auth cookies"))
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

	if err := h.setAuthCookies(c, pair); err != nil {
		_ = c.Error(apierr.Internal("failed to set auth cookies"))
		return
	}
	c.JSON(http.StatusOK, pair)
}

// Logout clears the auth and CSRF cookies, effectively ending the browser session.
// It is intentionally idempotent — always returns 204 regardless of whether
// cookies were present, so the client can call it safely even when already
// logged out.
func (h *AuthHandler) Logout(c *gin.Context) {
	secure := h.production
	// Expire all auth cookies immediately by setting maxAge = -1.
	c.SetCookie(accessTokenCookieName, "", -1, "/", "", secure, true)
	c.SetCookie(refreshTokenCookieName, "", -1, "/v1/auth", "", secure, true)
	c.SetCookie("csrf_token", "", -1, "/", "", secure, false)
	c.Status(http.StatusNoContent)
}
