package err

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ValidationFieldError represents a single field validation error.
type ValidationFieldError struct {
	Field         string `json:"field"`
	RejectedValue any    `json:"rejected_value"`
	Message       string `json:"message"`
}

// TranslateValidationErrors converts go-playground/validator.ValidationErrors
// into a slice of human-readable ValidationFieldError.
func TranslateValidationErrors(err error) []ValidationFieldError {
	ve, ok := err.(validator.ValidationErrors)
	if !ok {
		return nil
	}

	out := make([]ValidationFieldError, 0, len(ve))
	for _, fe := range ve {
		field := strings.ToLower(fe.Field())
		rejectedValue := fe.Value()

		var message string
		switch fe.Tag() {
		case "required":
			message = fmt.Sprintf("%s is required", field)
		case "email":
			message = fmt.Sprintf("%s must be a valid email", field)
		case "min":
			message = fmt.Sprintf("%s must be at least %s", field, fe.Param())
		case "max":
			message = fmt.Sprintf("%s must be at most %s", field, fe.Param())
		case "len":
			message = fmt.Sprintf("%s must be exactly %s characters", field, fe.Param())
		case "oneof":
			message = fmt.Sprintf("%s must be one of [%s]", field, fe.Param())
		default:
			message = fmt.Sprintf("%s failed %s validation", field, fe.Tag())
		}

		out = append(out, ValidationFieldError{
			Field:         field,
			RejectedValue: rejectedValue,
			Message:       message,
		})
	}

	return out
}
