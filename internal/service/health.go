package service

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

// Pinger is the minimal interface required by HealthService to check database connectivity.
type Pinger interface {
	PingContext(ctx context.Context) error
}

const pingInterval = 15 * time.Second

// HealthService periodically pings the database and caches the result in memory.
type HealthService struct {
	mu         sync.RWMutex
	databaseOK bool
	db         Pinger
}

// NewHealthService creates a HealthService with the given Pinger dependency.
func NewHealthService(db Pinger) *HealthService {
	return &HealthService{db: db}
}

// DatabaseOK returns the cached database health state (read-lock protected).
func (s *HealthService) DatabaseOK() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.databaseOK
}

// setDatabaseOK updates the cached state (write-lock protected).
func (s *HealthService) setDatabaseOK(ok bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.databaseOK = ok
}

// Start performs an immediate ping, then launches a background goroutine
// that pings the database every pingInterval.
func (s *HealthService) Start() {
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
func (s *HealthService) ping() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := s.db.PingContext(ctx); err != nil {
		slog.Warn("health service: database ping failed", "error", err)
		s.setDatabaseOK(false)
		return
	}

	s.setDatabaseOK(true)
}

// SetDatabaseOKForTest exposes setDatabaseOK for use in tests across packages.
func (s *HealthService) SetDatabaseOKForTest(ok bool) {
	s.setDatabaseOK(ok)
}
