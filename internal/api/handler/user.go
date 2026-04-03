package handler

import (
	"errors"
	"net/http"
	"time"

	"i9pay/internal/api/dto"
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
)

// UserHandler handles user management HTTP endpoints.
type UserHandler struct {
	userSvc *service.UserService
}

// NewUserHandler creates a UserHandler.
func NewUserHandler(userSvc *service.UserService) *UserHandler {
	return &UserHandler{userSvc: userSvc}
}

// RegisterRoutes registers the user handler routes.
func (h *UserHandler) RegisterRoutes(rg *gin.RouterGroup) {
	users := rg.Group("/users")
	users.POST("", middleware.RequireRole(domain.RoleOwner), h.Create)
	users.GET("", middleware.RequireRole(domain.RoleAdmin), h.List)
	users.GET("/:user_id", middleware.RequireRole(domain.RoleAdmin), h.GetByID)
	users.PUT("/:user_id", middleware.RequireRole(domain.RoleOwner), h.Update)
	users.DELETE("/:user_id", middleware.RequireRole(domain.RoleOwner), h.Delete)
}

type userResponse struct {
	ID        string      `json:"id"`
	Email     string      `json:"email"`
	Name      string      `json:"name"`
	Role      domain.Role `json:"role"`
	CreatedAt *time.Time  `json:"created_at,omitempty"`
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{
		ID:        u.ID,
		Email:     u.Email,
		Name:      u.Name,
		Role:      u.Role,
		CreatedAt: nil,
	}
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

// Create handles POST /v1/users.
func (h *UserHandler) Create(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	user, err := h.userSvc.Create(c.Request.Context(), tid.String(), service.CreateUserRequest{
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

	c.JSON(http.StatusCreated, toUserResponse(user))
}

// List handles GET /v1/users.
func (h *UserHandler) List(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	users, err := h.userSvc.List(c.Request.Context(), tid.String())
	if err != nil {
		_ = c.Error(err)
		return
	}

	resp := make([]userResponse, len(users))
	for i := range users {
		resp[i] = toUserResponse(&users[i])
	}

	c.JSON(http.StatusOK, resp)
}

// GetByID handles GET /v1/users/:user_id.
func (h *UserHandler) GetByID(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	userID := c.Param("user_id")

	user, err := h.userSvc.GetByID(c.Request.Context(), tid.String(), userID)
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			_ = c.Error(apierr.NotFound(err.Error()))
			return
		}
		_ = c.Error(err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// Update handles PUT /v1/users/:user_id.
func (h *UserHandler) Update(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	userID := c.Param("user_id")

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	user, err := h.userSvc.Update(c.Request.Context(), tid.String(), userID, service.UpdateUserRequest{
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

	c.JSON(http.StatusOK, toUserResponse(user))
}

// Delete handles DELETE /v1/users/:user_id.
func (h *UserHandler) Delete(c *gin.Context) {
	tid, ok := dto.OrganizationID(c)
	if !ok {
		return
	}

	userID := c.Param("user_id")

	err := h.userSvc.Delete(c.Request.Context(), tid.String(), userID)
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
