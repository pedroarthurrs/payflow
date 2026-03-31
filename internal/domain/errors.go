package domain

import "errors"

// Sentinel errors returned by the domain layer.
var (
	ErrInvalidCredentials   = errors.New("invalid credentials")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrCustomerNotFound     = errors.New("customer not found")
	ErrAddressNotFound      = errors.New("address not found")
	ErrUnauthorized         = errors.New("invalid credentials")
	ErrBadRequest           = errors.New("organization_id, email and password or api_key required")
	ErrNotFound             = errors.New("not found")
	ErrConflict             = errors.New("conflict")
	ErrForbidden            = errors.New("forbidden")
)
