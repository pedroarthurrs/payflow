package err

import (
	"errors"
	"net/http"
)

// AppError is a typed application error that carries an HTTP status and a
// user-facing message. Handlers create these and attach them to the context
// via c.Error(); the Errors middleware is responsible for writing the response.
type AppError struct {
	Status  int
	Message string
}

func (e *AppError) Error() string { return e.Message }

// As unwraps err into *AppError. Returns (nil, false) if the error chain does
// not contain an *AppError.
func As(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}

// BadRequest creates a 400 Bad Request error.
func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: msg}
}

// Unauthorized creates a 401 Unauthorized error.
func Unauthorized(msg string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: msg}
}

// Forbidden creates a 403 Forbidden error.
func Forbidden(msg string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: msg}
}

// NotFound creates a 404 Not Found error.
func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: msg}
}

// Conflict creates a 409 Conflict error.
func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: msg}
}

// Internal creates a 500 Internal Server Error.
func Internal(msg string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: msg}
}

// ServiceUnavailable creates a 503 Service Unavailable error.
func ServiceUnavailable(msg string) *AppError {
	return &AppError{Status: http.StatusServiceUnavailable, Message: msg}
}
