package dto

import (
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// TenantID extracts the tenant UUID from the JWT claims in the Gin context.
// Uses safe type assertion to avoid panics.
func TenantID(c *gin.Context) (uuid.UUID, bool) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(apierr.Internal("could not parse claims"))
		return uuid.UUID{}, false
	}
	tid, err := uuid.Parse(claims.TenantID)
	if err != nil {
		_ = c.Error(apierr.Internal("invalid tenant id in claims"))
		return uuid.UUID{}, false
	}
	return tid, true
}

// CustomerID parses the ":customer_id" path parameter as a UUID.
func CustomerID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("customer_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid customer id"))
		return uuid.UUID{}, false
	}
	return id, true
}
