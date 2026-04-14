package domain

import "errors"

// Sentinel errors returned by the domain layer.
var (
	ErrOrganizationNotFound    = errors.New("organization not found")
	ErrCustomerNotFound        = errors.New("customer not found")
	ErrProductNotFound         = errors.New("product not found")
	ErrAddressNotFound         = errors.New("address not found")
	ErrCredentialNotFound      = errors.New("credential not found")
	ErrCredentialLogNotFound   = errors.New("credential log not found")
	ErrUnauthorized            = errors.New("invalid credentials")
	ErrBadRequest              = errors.New("organization_id, email and password or api_key required")
	ErrNotFound                = errors.New("not found")
	ErrConflict                = errors.New("conflict")
	ErrForbidden               = errors.New("forbidden")
	ErrInternal                = errors.New("internal server error")
	ErrDuplicateEmail          = errors.New("customer with this email already exists")
	ErrDuplicateDocument       = errors.New("customer with this document already exists")
	ErrWebhookEndpointNotFound = errors.New("webhook endpoint not found")
	ErrWebhookLogNotFound      = errors.New("webhook log not found")
	ErrPlanNotFound            = errors.New("plan not found")
	ErrPlanInUse               = errors.New("plan is referenced by active subscriptions")

	// Subscription errors
	ErrSubscriptionNotFound          = errors.New("subscription not found")
	ErrInvalidSubscriptionTransition = errors.New("invalid subscription status transition")
	ErrSubscriptionCanceled          = errors.New("subscription is already canceled")
	ErrNoSubscriptionItems           = errors.New("subscription must have at least one item")
	ErrScheduleAlreadyPending        = errors.New("subscription already has a pending schedule")
	ErrScheduleNotFound              = errors.New("subscription schedule not found")

	// Gateway errors
	ErrGatewayChargeFailed = errors.New("gateway charge failed")
	ErrGatewayRefundFailed = errors.New("gateway refund failed")
	ErrGatewayCancelFailed = errors.New("gateway cancel failed")
	ErrGatewayUnavailable  = errors.New("gateway unavailable")

	// Recurrence errors
	ErrSubscriptionAlreadyLocked = errors.New("subscription billing is already locked")
	ErrBillingAmountZero         = errors.New("subscription billing amount is zero")
)
