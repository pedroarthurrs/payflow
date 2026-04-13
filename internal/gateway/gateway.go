package gateway

import (
	"i9pay/internal/domain"
)

// New returns the appropriate Gateway implementation based on the provider name.
// Returns nil if the provider name is not recognized.
func New(providerName, apiKey string) domain.Gateway {
	switch providerName {
	case "stripe":
		return NewStripeAdapter(apiKey)
	default:
		return nil
	}
}
