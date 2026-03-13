package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTenantNotFound     = errors.New("tenant not found")
)
