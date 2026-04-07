package config

// Flag identifies a feature flag by name.
type Flag string

// Predefined feature flags.
// Each flag maps to a Docker-injected environment variable: FEATURE_<FLAG_NAME>
// (e.g. FEATURE_MAINTENANCE=true). Values are set in the service's docker-compose
// or container definition — no .env file is used.
const (
	// FlagMaintenance enables maintenance mode, blocking non-health endpoints.
	FlagMaintenance Flag = "MAINTENANCE"
)

// FlagProvider abstracts the feature flag backing store.
// Implementations can use environment variables, AWS AppConfig, a database,
// or any remote config service without changing call sites.
type FlagProvider interface {
	// IsEnabled reports whether the named flag is active.
	// Implementations must be safe for concurrent use.
	IsEnabled(flag Flag) bool

	// Reload fetches the latest values from the backing store.
	// For env-based providers this re-reads os.Getenv; for remote providers
	// it performs a network fetch. Safe to call on a schedule or on demand.
	Reload() error
}
