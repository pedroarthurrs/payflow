package dto

import (
	apierr "i9pay/internal/api/err"
	"i9pay/internal/api/middleware"
	"i9pay/internal/auth/token"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// OrganizationID extracts the organization UUID from the JWT claims in the Gin context.
// Uses safe type assertion to avoid panics.
func OrganizationID(c *gin.Context) (uuid.UUID, bool) {
	claims, ok := c.MustGet(middleware.ClaimsKey).(*token.Claims)
	if !ok {
		_ = c.Error(apierr.Internal("could not parse claims"))
		return uuid.UUID{}, false
	}
	tid, err := uuid.Parse(claims.OrganizationID)
	if err != nil {
		_ = c.Error(apierr.Internal("invalid organization id in claims"))
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

// ProductID parses the ":product_id" path parameter as a UUID.
func ProductID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("product_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid product id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// CredentialID parses the ":credential_id" path parameter as a UUID.
func CredentialID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("credential_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid credential id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// LogID parses the ":log_id" path parameter as a UUID.
func LogID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("log_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid log id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// WebhookID parses the ":webhook_id" path parameter as a UUID.
func WebhookID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("webhook_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid webhook id"))
		return uuid.UUID{}, false
	}
	return id, true
}
