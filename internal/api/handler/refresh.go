package handler

import (
	apierr "i9pay/internal/api/err"
	i9token "i9pay/internal/token"
	"net/http"

	"github.com/gin-gonic/gin"
)

// RefreshHandler handles access token renewal using a valid refresh token.
type RefreshHandler struct {
	svc *i9token.Service
}

// NewRefreshHandler creates a RefreshHandler backed by the given token Service.
func NewRefreshHandler(svc *i9token.Service) *RefreshHandler {
	return &RefreshHandler{svc: svc}
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Post validates the supplied refresh token and returns a new access/refresh
// token pair. The old refresh token is immediately superseded; clients should
// replace it with the one returned in the response.
func (h *RefreshHandler) Post(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		_ = c.Error(apierr.BadRequest(err.Error()))
		return
	}

	pair, err := h.svc.Refresh(req.RefreshToken)
	if err != nil {
		_ = c.Error(apierr.Unauthorized("invalid or expired refresh token"))
		return
	}

	c.JSON(http.StatusOK, pair)
}
