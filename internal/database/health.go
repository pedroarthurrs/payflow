package database

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Pinger is the minimal interface required by HealthChecker to check database connectivity.
type Pinger interface {
	PingContext(ctx context.Context) error
}

const pingInterval = 15 * time.Second

// HealthChecker periodically pings the database and caches the result in memory.
type HealthChecker struct {
	mu       sync.RWMutex
	dbStatus bool
	db       Pinger
}

// NewHealthChecker creates a HealthChecker with the given Pinger dependency.
func NewHealthChecker(db Pinger) *HealthChecker {
	return &HealthChecker{db: db}
}

// DatabaseOK returns the cached database health state (read-lock protected).
func (s *HealthChecker) DatabaseOK() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dbStatus
}

// updateDBStatus updates the cached state (write-lock protected).
func (s *HealthChecker) updateDBStatus(ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.dbStatus = ok
}

// Start performs an immediate ping, then launches a background goroutine
// that pings the database every pingInterval.
func (s *HealthChecker) Start() {
	s.ping()

	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for range ticker.C {
			s.ping()
		}
	}()
}

// ping executes a single database ping with a 2-second timeout.
func (s *HealthChecker) ping() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		slog.Warn("health checker: database ping failed", "error", err)
		s.updateDBStatus(false)
		return
	}

	s.updateDBStatus(true)
}

// SetDatabaseOKForTest exposes updateDBStatus for use in tests across packages.
func (s *HealthChecker) SetDatabaseOKForTest(ok bool) {
	s.updateDBStatus(ok)
}
