package handler

import (
	"i9pay/internal/api/middleware"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type LoginHandler struct {
	jwtSecret []byte
}

func NewLoginHandler(jwtSecret string) *LoginHandler {
	return &LoginHandler{jwtSecret: []byte(jwtSecret)}
}

type loginRequest struct {
	Name     string `json:"name" binding:"required"`
	Password string `json:"password"    binding:"required"`
}

func (h *LoginHandler) Post(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	now := time.Now()
	claims := middleware.Claims{
		Name:     req.Name,
		Password: req.Password,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    "i9pay",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(24 * time.Hour)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(h.jwtSecret)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "could not generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token":      signed,
		"expires_at": claims.ExpiresAt.Time,
	})
}
