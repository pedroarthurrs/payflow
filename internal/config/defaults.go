package config

import "time"

// Operational defaults — tuning parameters that do not vary per environment
// and are not sensitive. Defined as constants so they are auditable in code
// and do not require environment variable management.

// Rate limiting — public IP routes.
const (
	RateLimitWindow      = 60 * time.Second
	RateLimitMaxRequests = 100
)

// Rate limiting — login endpoint.
const (
	RateLimitLoginWindow      = 300 * time.Second
	RateLimitLoginMaxRequests = 5
)

// Rate limiting — authenticated organization routes.
const (
	RateLimitOrganizationWindow      = 60 * time.Second
	RateLimitOrganizationMaxRequests = 200
)

// Webhook worker.
const (
	WebhookQueueBufferSize = 1000
	WebhookWorkerCount     = 5
)

// Recurrence engine.
const (
	RecurrenceQueueBufferSize = 500
	RecurrenceWorkerCount     = 3
	RecurrenceScanCron        = "* * * * *"
	RecurrenceMaxAttempts     = 4
	RecurrenceBatchSize       = 100
)
