package domain

import "errors"

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrTenantNotFound     = errors.New("tenant not found")
	ErrCustomerNotFound   = errors.New("customer not found")
	ErrAddressNotFound    = errors.New("address not found")
)
