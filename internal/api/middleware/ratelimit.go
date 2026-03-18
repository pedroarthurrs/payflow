package middleware

import (
	"net/http"
	"strconv"
	"sync"
	"time"

	"i9pay/internal/token"

	"github.com/gin-gonic/gin"
)

// visitor tracks request count and window expiry for a single key (IP or tenant_id).
type visitor struct {
	count     int
	expiresAt time.Time
}

// RateLimiterStore is an in-memory store protected by sync.Mutex.
// Keys are IP addresses or tenant IDs. Expired entries are lazily cleaned.
type RateLimiterStore struct {
	mu       sync.Mutex
	visitors map[string]*visitor
}

// NewRateLimiterStore creates a new in-memory rate limiter store.
func NewRateLimiterStore() *RateLimiterStore {
	return &RateLimiterStore{
		visitors: make(map[string]*visitor),
	}
}

// Allow checks if the key is within the rate limit.
// Returns (allowed bool, remaining int, resetAt time.Time).
func (s *RateLimiterStore) Allow(key string, maxRequests int, window time.Duration) (allowed bool, remaining int, resetAt time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()

	v, exists := s.visitors[key]
	if !exists || v.expiresAt.Before(now) {
		v = &visitor{
			count:     1,
			expiresAt: now.Add(window),
		}
		s.visitors[key] = v
		return true, maxRequests - 1, v.expiresAt
	}

	v.count++

	if v.count > maxRequests {
		return false, 0, v.expiresAt
	}

	return true, maxRequests - v.count, v.expiresAt
}

// RateLimitByIP returns a Gin middleware that rate-limits by client IP.
func RateLimitByIP(store *RateLimiterStore, maxRequests int, window time.Duration, rejectMsg string) gin.HandlerFunc {
	return func(c *gin.Context) {
		ip := c.ClientIP()
		allowed, remaining, resetAt := store.Allow(ip, maxRequests, window)

		c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
			c.Header("Retry-After", strconv.Itoa(int(time.Until(resetAt).Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": rejectMsg})
			return
		}

		c.Next()
	}
}

// RateLimitByTenant returns a Gin middleware that rate-limits by tenant_id
// extracted from JWT claims stored in the Gin context by the Auth middleware.
func RateLimitByTenant(store *RateLimiterStore, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims, exists := c.Get(ClaimsKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
			return
		}
		tenantID := claims.(*token.Claims).TenantID

		allowed, remaining, resetAt := store.Allow(tenantID, maxRequests, window)

		c.Header("X-RateLimit-Limit", strconv.Itoa(maxRequests))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))

		if !allowed {
			c.Header("X-RateLimit-Reset", strconv.FormatInt(resetAt.Unix(), 10))
			c.Header("Retry-After", strconv.Itoa(int(time.Until(resetAt).Seconds())+1))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{"error": "too many requests, try again later"})
			return
		}

		c.Next()
	}
}
