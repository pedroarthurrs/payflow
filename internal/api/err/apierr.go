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

func BadRequest(msg string) *AppError {
	return &AppError{Status: http.StatusBadRequest, Message: msg}
}

func Unauthorized(msg string) *AppError {
	return &AppError{Status: http.StatusUnauthorized, Message: msg}
}

func Forbidden(msg string) *AppError {
	return &AppError{Status: http.StatusForbidden, Message: msg}
}

func NotFound(msg string) *AppError {
	return &AppError{Status: http.StatusNotFound, Message: msg}
}

func Conflict(msg string) *AppError {
	return &AppError{Status: http.StatusConflict, Message: msg}
}

func Internal(msg string) *AppError {
	return &AppError{Status: http.StatusInternalServerError, Message: msg}
}

func ServiceUnavailable(msg string) *AppError {
	return &AppError{Status: http.StatusServiceUnavailable, Message: msg}
}
