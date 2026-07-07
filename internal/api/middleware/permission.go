package middleware

import (
	"context"
	"net/http"
	"sync"
	"time"

	i9token "payflow/internal/auth/token"
	"payflow/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// PermissionResolver is a thin interface that allows RequirePermission to use
// either the real PermissionService or a test double.
type PermissionResolver interface {
	HasPermission(ctx context.Context, orgID, userAccountID uuid.UUID, perm domain.Permission) (bool, error)
}

// permCacheKey identifies a cached permission result.
type permCacheKey struct {
	orgID         uuid.UUID
	userAccountID uuid.UUID
	perm          domain.Permission
}

type permCacheEntry struct {
	allowed   bool
	expiresAt time.Time
}

// PermissionCache is a TTL-based in-memory cache for effective permission results.
// Safe for concurrent use.
type PermissionCache struct {
	mu      sync.Mutex
	entries map[permCacheKey]permCacheEntry
	ttl     time.Duration
}

// NewPermissionCache creates a PermissionCache with the given TTL.
func NewPermissionCache(ttl time.Duration) *PermissionCache {
	return &PermissionCache{
		entries: make(map[permCacheKey]permCacheEntry),
		ttl:     ttl,
	}
}

func (c *PermissionCache) get(key permCacheKey) (allowed, found bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiresAt) {
		delete(c.entries, key)
		return false, false
	}
	return e.allowed, true
}

func (c *PermissionCache) set(key permCacheKey, allowed bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = permCacheEntry{allowed: allowed, expiresAt: time.Now().Add(c.ttl)}
}

// CachingPermissionResolver wraps a PermissionResolver and caches results with a 5-minute TTL.
type CachingPermissionResolver struct {
	inner PermissionResolver
	cache *PermissionCache
}

// NewCachingPermissionResolver creates a CachingPermissionResolver with a 5-minute TTL.
func NewCachingPermissionResolver(inner PermissionResolver) *CachingPermissionResolver {
	return &CachingPermissionResolver{
		inner: inner,
		cache: NewPermissionCache(5 * time.Minute),
	}
}

// HasPermission checks the cache first; on a miss it delegates to the inner resolver
// and caches the result for the configured TTL.
func (r *CachingPermissionResolver) HasPermission(ctx context.Context, orgID, userAccountID uuid.UUID, perm domain.Permission) (bool, error) {
	key := permCacheKey{orgID: orgID, userAccountID: userAccountID, perm: perm}
	if allowed, ok := r.cache.get(key); ok {
		return allowed, nil
	}
	allowed, err := r.inner.HasPermission(ctx, orgID, userAccountID, perm)
	if err == nil {
		r.cache.set(key, allowed)
	}
	return allowed, err
}

// RequirePermission returns a Gin middleware that checks if the authenticated user
// has the specified effective permission in their organization.
//
// Unlike RequireRole (which only checks role hierarchy), this middleware performs
// a DB lookup via PermissionResolver and accounts for individual overrides stored
// in member_permissions (granted=true/false).
//
//   - If Claims are absent → 401
//   - If token is system (no user_id) → 403 "user role required"
//   - If organization_id or user_id are invalid → 403
//   - If user does not have the effective permission → 403 "insufficient permissions"
func RequirePermission(resolver PermissionResolver, perm domain.Permission) gin.HandlerFunc {
	return func(c *gin.Context) {
		val, exists := c.Get(ClaimsKey)
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		claims, ok := val.(*i9token.Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization required"})
			return
		}

		if claims.UserID == "" {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "user role required"})
			return
		}

		orgID, err := uuid.Parse(claims.OrganizationID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid organization"})
			return
		}

		userAccountID, err := uuid.Parse(claims.UserID)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "invalid user"})
			return
		}

		allowed, err := resolver.HasPermission(c.Request.Context(), orgID, userAccountID, perm)
		if err != nil || !allowed {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "insufficient permissions"})
			return
		}

		c.Next()
	}
}
