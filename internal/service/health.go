package service

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"i9pay/internal/database"
)

const pingInterval = 15 * time.Second

// HealthService periodically pings the database and caches the result in memory.
type HealthService struct {
	mu         sync.RWMutex
	databaseOK bool
}

// NewHealthService creates a HealthService with databaseOK initialized to false.
func NewHealthService() *HealthService {
	return &HealthService{}
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
// that pings the database every 15 seconds.
func (s *HealthService) Start(db *database.DB) {
	s.ping(db)

	go func() {
		ticker := time.NewTicker(pingInterval)
		defer ticker.Stop()

		for range ticker.C {
			s.ping(db)
		}
	}()
}

// ping executes a single database ping with a 2-second timeout.
func (s *HealthService) ping(db *database.DB) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
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
