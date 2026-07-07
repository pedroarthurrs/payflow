package dto

import (
	apierr "payflow/internal/api/err"
	"payflow/internal/api/middleware"
	"payflow/internal/auth/token"

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

// PaymentID parses the ":payment_id" path parameter as a UUID.
func PaymentID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("payment_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid payment id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// PlanID parses the ":plan_id" path parameter as a UUID.
func PlanID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("plan_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid plan id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// SubscriptionID parses the ":subscription_id" path parameter as a UUID.
func SubscriptionID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("subscription_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid subscription id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// ScheduleID parses the ":schedule_id" path parameter as a UUID.
func ScheduleID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("schedule_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid schedule id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// WalletID parses the ":wallet_id" path parameter as a UUID.
func WalletID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("wallet_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid wallet id"))
		return uuid.UUID{}, false
	}
	return id, true
}

// InvoiceID parses the ":invoice_id" path parameter as a UUID.
func InvoiceID(c *gin.Context) (uuid.UUID, bool) {
	id, err := uuid.Parse(c.Param("invoice_id"))
	if err != nil {
		_ = c.Error(apierr.BadRequest("invalid invoice id"))
		return uuid.UUID{}, false
	}
	return id, true
}
