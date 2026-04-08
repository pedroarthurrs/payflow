package middleware

import (
	"errors"
	"net/http"

	apierr "i9pay/internal/api/err"
	"i9pay/internal/domain"
	"i9pay/internal/monitor"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// domainStatusMap maps domain sentinel errors to HTTP status codes.
var domainStatusMap = map[error]int{
	domain.ErrNotFound:                http.StatusNotFound,
	domain.ErrCustomerNotFound:        http.StatusNotFound,
	domain.ErrAddressNotFound:         http.StatusNotFound,
	domain.ErrOrganizationNotFound:    http.StatusNotFound,
	domain.ErrConflict:                http.StatusConflict,
	domain.ErrDuplicateEmail:          http.StatusConflict,
	domain.ErrDuplicateDocument:       http.StatusConflict,
	domain.ErrUnauthorized:            http.StatusUnauthorized,
	domain.ErrForbidden:               http.StatusForbidden,
	domain.ErrBadRequest:              http.StatusBadRequest,
	domain.ErrInternal:                http.StatusInternalServerError,
	domain.ErrWebhookEndpointNotFound: http.StatusNotFound,
	domain.ErrWebhookLogNotFound:      http.StatusNotFound,
	domain.ErrTransactionNotFound:     http.StatusNotFound,            // 404
	domain.ErrTerminalState:           http.StatusConflict,            // 409
	domain.ErrSameStateTransition:     http.StatusUnprocessableEntity, // 422
	domain.ErrInvalidTransition:       http.StatusUnprocessableEntity, // 422
}

// Errors reads errors attached via c.Error() after the handler chain runs and
// writes a uniform JSON response. Skips if the response was already written
// (e.g. by AbortWithStatusJSON in other middlewares).
func Errors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Written() || len(c.Errors) == 0 {
			return
		}

		err := c.Errors.Last().Err

		// Check for validation errors first
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			c.JSON(http.StatusBadRequest, gin.H{"errors": apierr.TranslateValidationErrors(err)})
			return
		}

		if appErr, ok := apierr.As(err); ok {
			c.JSON(appErr.Status, gin.H{"error": appErr.Message})
			return
		}

		// Check domain sentinel errors
		for sentinel, status := range domainStatusMap {
			if errors.Is(err, sentinel) {
				c.JSON(status, gin.H{"error": sentinel.Error()})
				return
			}
		}

		// Unexpected error — report to monitor and return 500.
		monitor.CaptureException(c, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	}
}

// Recovery catches panics in the handler chain and returns a 500 response
// instead of crashing the server.
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, _ any) {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
	})
}
