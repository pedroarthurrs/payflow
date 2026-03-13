package middleware

import (
	"errors"
	apierr "i9pay/internal/api/err"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

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
