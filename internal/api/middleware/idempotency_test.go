package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// mockIdempotencyStorage is a test double for domain.IdempotencyStorage.
type mockIdempotencyStorage struct {
	checkFn func(ctx context.Context, key string, orgID uuid.UUID) (*domain.IdempotencyRecord, error)
	saveFn  func(ctx context.Context, key string, orgID uuid.UUID, requestPath string, record domain.IdempotencyRecord, ttl time.Duration) error
}

var _ domain.IdempotencyStorage = (*mockIdempotencyStorage)(nil)

func (m *mockIdempotencyStorage) Check(ctx context.Context, key string, orgID uuid.UUID) (*domain.IdempotencyRecord, error) {
	if m.checkFn != nil {
		return m.checkFn(ctx, key, orgID)
	}
	return nil, nil
}

func (m *mockIdempotencyStorage) Save(ctx context.Context, key string, orgID uuid.UUID, requestPath string, record domain.IdempotencyRecord, ttl time.Duration) error {
	if m.saveFn != nil {
		return m.saveFn(ctx, key, orgID, requestPath, record, ttl)
	}
	return nil
}

// setupIdempotencyRouter creates a Gin engine with Auth claims and the Idempotency middleware.
func setupIdempotencyRouter(storage domain.IdempotencyStorage, orgID uuid.UUID) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ClaimsKey, &token.Claims{
			OrganizationID: orgID.String(),
			AuthType:       "user",
		})
		c.Next()
	})
	r.Use(Idempotency(storage))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"id": "new-resource"})
	})
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})
	r.PATCH("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"updated": true})
	})
	return r
}

func TestIdempotency_CacheHit_ReplaysCachedResponse(t *testing.T) {
	orgID := uuid.New()
	cachedBody, _ := json.Marshal(gin.H{"id": "cached-resource"})

	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, key string, gotOrgID uuid.UUID) (*domain.IdempotencyRecord, error) {
			if key != "test-key" || gotOrgID != orgID {
				t.Fatalf("unexpected check args: key=%q orgID=%v", key, gotOrgID)
			}
			return &domain.IdempotencyRecord{
				ResponseCode: http.StatusCreated,
				ResponseBody: cachedBody,
			}, nil
		},
	}

	r := setupIdempotencyRouter(storage, orgID)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	req.Header.Set(IdempotencyHeader, "test-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	if w.Body.String() != string(cachedBody) {
		t.Fatalf("expected body %q, got %q", string(cachedBody), w.Body.String())
	}
}

func TestIdempotency_CacheMiss_CallsHandlerAndSaves(t *testing.T) {
	orgID := uuid.New()
	var savedRecord domain.IdempotencyRecord
	var savedKey string

	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, _ string, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
			return nil, nil
		},
		saveFn: func(_ context.Context, key string, _ uuid.UUID, _ string, record domain.IdempotencyRecord, ttl time.Duration) error {
			savedKey = key
			savedRecord = record
			if ttl != DefaultIdempotencyTTL {
				t.Fatalf("expected ttl %v, got %v", DefaultIdempotencyTTL, ttl)
			}
			return nil
		},
	}

	r := setupIdempotencyRouter(storage, orgID)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	req.Header.Set(IdempotencyHeader, "new-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
	if savedKey != "new-key" {
		t.Fatalf("expected saved key %q, got %q", "new-key", savedKey)
	}
	if savedRecord.ResponseCode != http.StatusCreated {
		t.Fatalf("expected saved code %d, got %d", http.StatusCreated, savedRecord.ResponseCode)
	}
}

func TestIdempotency_NoHeader_SkipsMiddleware(t *testing.T) {
	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, _ string, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
			t.Fatal("check should not be called without header")
			return nil, nil
		},
	}

	r := setupIdempotencyRouter(storage, uuid.New())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}

func TestIdempotency_GETRequest_SkipsMiddleware(t *testing.T) {
	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, _ string, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
			t.Fatal("check should not be called on GET")
			return nil, nil
		},
	}

	r := setupIdempotencyRouter(storage, uuid.New())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/test", http.NoBody)
	req.Header.Set(IdempotencyHeader, "some-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestIdempotency_PATCHRequest_IsIntercepted(t *testing.T) {
	orgID := uuid.New()
	cachedBody, _ := json.Marshal(gin.H{"updated": true})

	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, _ string, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
			return &domain.IdempotencyRecord{
				ResponseCode: http.StatusOK,
				ResponseBody: cachedBody,
			}, nil
		},
	}

	r := setupIdempotencyRouter(storage, orgID)
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPatch, "/test", http.NoBody)
	req.Header.Set(IdempotencyHeader, "patch-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, w.Code)
	}
	if w.Body.String() != string(cachedBody) {
		t.Fatalf("expected body %q, got %q", string(cachedBody), w.Body.String())
	}
}

func TestIdempotency_StorageCheckError_ProceedsToHandler(t *testing.T) {
	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, _ string, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
			return nil, domain.ErrInternal
		},
	}

	r := setupIdempotencyRouter(storage, uuid.New())
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	req.Header.Set(IdempotencyHeader, "err-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d on storage error, got %d", http.StatusCreated, w.Code)
	}
}

func TestIdempotency_NoClaims_SkipsMiddleware(t *testing.T) {
	storage := &mockIdempotencyStorage{
		checkFn: func(_ context.Context, _ string, _ uuid.UUID) (*domain.IdempotencyRecord, error) {
			t.Fatal("check should not be called without claims")
			return nil, nil
		},
	}

	gin.SetMode(gin.TestMode)
	r := gin.New()
	// No claims middleware
	r.Use(Idempotency(storage))
	r.POST("/test", func(c *gin.Context) {
		c.JSON(http.StatusCreated, gin.H{"ok": true})
	})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/test", http.NoBody)
	req.Header.Set(IdempotencyHeader, "some-key")
	w := httptest.NewRecorder()

	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, w.Code)
	}
}
