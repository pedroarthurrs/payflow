package config

import (
	"os"
	"strings"
	"sync"
)

// EnvFlagProvider implements FlagProvider using Docker-injected environment variables.
// Flag values are resolved as FEATURE_<FLAG_NAME> (upper-cased).
// "true", "1", and "yes" (case-insensitive) are treated as enabled.
//
// This is the default provider. To switch to AWS AppConfig or another remote
// source, implement FlagProvider and inject it wherever EnvFlagProvider is used —
// no other code changes are needed.
type EnvFlagProvider struct {
	mu       sync.RWMutex
	known    []Flag
	resolved map[Flag]bool
}

// NewEnvFlagProvider creates an EnvFlagProvider registered with the given flags
// and performs an initial load from the current environment.
func NewEnvFlagProvider(flags ...Flag) *EnvFlagProvider {
	p := &EnvFlagProvider{
		known:    flags,
		resolved: make(map[Flag]bool, len(flags)),
	}
	p.load()
	return p
}

// IsEnabled implements FlagProvider.
func (p *EnvFlagProvider) IsEnabled(flag Flag) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()
	return p.resolved[flag]
}

// Reload re-reads the environment variables for all registered flags.
// It always returns nil; the error return exists to satisfy FlagProvider
// so that remote providers can propagate fetch errors.
func (p *EnvFlagProvider) Reload() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.load()
	return nil
}

// load must be called with the write lock held (or before the provider is shared).
func (p *EnvFlagProvider) load() {
	for _, f := range p.known {
		key := "FEATURE_" + string(f)
		val := strings.ToLower(strings.TrimSpace(os.Getenv(key)))
		p.resolved[f] = val == "true" || val == "1" || val == "yes"
	}
}
