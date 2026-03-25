package user

import (
	"errors"
	"net/http"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"
	"i9pay/internal/token"

	"github.com/gin-gonic/gin"
)

// Handler handles user management HTTP endpoints.
type Handler struct {
	userSvc *service.UserService
}

// NewHandler creates a user Handler.
func NewHandler(userSvc *service.UserService) *Handler {
	return &Handler{userSvc: userSvc}
}

type createUserRequest struct {
	Name     string      `json:"name"     binding:"required"`
	Email    string      `json:"email"    binding:"required"`
	Password string      `json:"password" binding:"required"`
	Role     domain.Role `json:"role"     binding:"required"`
}

type updateUserRequest struct {
	Name string      `json:"name" binding:"required"`
	Role domain.Role `json:"role" binding:"required"`
}

// tenantIDFromClaims extracts the tenant ID string from JWT claims in context.
func tenantIDFromClaims(c *gin.Context) (string, bool) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(apierr.Internal("could not parse claims"))
		return "", false
	}
	return claims.TenantID, true
}

// Create handles POST /v1/users.
func (h *Handler) Create(c *gin.Context) {
	tid, ok := tenantIDFromClaims(c)
	if !ok {
		return
	}

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	user, err := h.userSvc.Create(c.Request.Context(), tid, service.CreateUserRequest{
		Name:     req.Name,
		Email:    req.Email,
		Password: req.Password,
		Role:     req.Role,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			_ = c.Error(apierr.Conflict(err.Error()))
			return
		}
		if errors.Is(err, domain.ErrForbidden) {
			_ = c.Error(apierr.Forbidden(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, user)
}

// List handles GET /v1/users.
func (h *Handler) List(c *gin.Context) {
	tid, ok := tenantIDFromClaims(c)
	if !ok {
		return
	}

	users, err := h.userSvc.List(c.Request.Context(), tid)
	if err != nil {
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, users)
}

// Update handles PUT /v1/users/:user_id.
func (h *Handler) Update(c *gin.Context) {
	tid, ok := tenantIDFromClaims(c)
	if !ok {
		return
	}

	userID := c.Param("user_id")

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	user, err := h.userSvc.Update(c.Request.Context(), tid, userID, service.UpdateUserRequest{
		Name: req.Name,
		Role: req.Role,
	})
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, user)
}

// Delete handles DELETE /v1/users/:user_id.
func (h *Handler) Delete(c *gin.Context) {
	tid, ok := tenantIDFromClaims(c)
	if !ok {
		return
	}

	userID := c.Param("user_id")

	err := h.userSvc.Delete(c.Request.Context(), tid, userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.Status(http.StatusNoContent)
}
