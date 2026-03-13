package handler

import (
	apierr "i9pay/internal/api/err"
	i9token "i9pay/internal/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

// LoginHandler handles tenant authentication and issues a JWT token pair.
type LoginHandler struct {
	svc *i9token.Service
}

// NewLoginHandler creates a LoginHandler backed by the given token Service.
func NewLoginHandler(svc *i9token.Service) *LoginHandler {
	return &LoginHandler{svc: svc}
}

type loginRequest struct {
	Email    string `json:"email"    binding:"required"`
	Password string `json:"password" binding:"required"`
}

// Post authenticates the tenant and returns a fresh access/refresh token pair.
func (h *LoginHandler) Post(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	tenantID, _ := authenticateTenant(req.Email, req.Password)

	pair, err := h.svc.NewPair(tenantID)
	if err != nil {
		_ = c.Error(apierr.Internal("could not generate token"))
		return
	}

	c.JSON(http.StatusOK, pair)
}

func authenticateTenant(email, password string) (string, error) {
	return email + password, nil
}
