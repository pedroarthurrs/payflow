package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"i9pay/internal/api/middleware"
	"i9pay/internal/domain"
	"i9pay/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// setupCredentialLogRouterWithRole creates a test router with a specific role injected.
func setupCredentialLogRouterWithRole(svc *mockCredentialLogService, orgID, role string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	repo := &credentialLogRepoAdapter{svc: svc}
	realSvc := service.NewCredentialService(nil, repo, nil, "test-pepper")
	h := NewCredentialHandler(realSvc)
	r := gin.New()
	r.Use(middleware.Errors())
	r.Use(withClaimsAndRole(orgID, role))

	logs := r.Group("/credentials/logs")
	logs.Use(middleware.RequireRole(domain.RoleAdmin))
	logs.GET("", h.ListLogs)
	logs.GET("/:log_id", h.GetLogByID)
	return r
}

// ── Property 9: Limit boundary validation ────────────────────────────────────

// TestProperty_CredentialLog_LimitBoundary verifies that any limit outside [1,100]
// returns 400 Bad Request.
// Feature: credential-logs, Property 9: Limit boundary validation
func TestProperty_CredentialLog_LimitBoundary(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a limit that is either < 1 or > 100.
		limit := rapid.OneOf(
			rapid.IntRange(-1000, 0),
			rapid.IntRange(101, 1000),
		).Draw(t, "limit")

		svc := &mockCredentialLogService{}
		r := setupCredentialLogRouter(svc, uuid.New().String())

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
			fmt.Sprintf("/credentials/logs?limit=%d", limit), http.NoBody)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("limit=%d: expected 400, got %d (body=%s)", limit, w.Code, w.Body.String())
		}
	})
}

// ── Property 10: Invalid cursor returns 400 ───────────────────────────────────

// TestProperty_CredentialLog_InvalidCursor verifies that any malformed cursor
// string returns 400 Bad Request.
// Feature: credential-logs, Property 10: Invalid cursor returns 400
func TestProperty_CredentialLog_InvalidCursor(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate strings that are URL-safe but not valid cursor tokens.
		badCursor := rapid.OneOf(
			// Random alphanumeric strings — valid URL chars but not valid base64url JSON cursors.
			rapid.StringMatching(`[a-zA-Z0-9_-]{1,20}`).Filter(func(s string) bool {
				// Exclude strings that happen to be valid cursors by accident.
				_, err := service.DecodeCursor(s)
				return err != nil
			}),
			// Valid base64url encoding of non-JSON content.
			rapid.Map(rapid.StringMatching(`[a-z]{5,20}`), func(s string) string {
				return base64.RawURLEncoding.EncodeToString([]byte(s))
			}),
			// Valid base64url of JSON missing required fields.
			rapid.Map(rapid.StringMatching(`[a-z]{3,10}`), func(s string) string {
				return base64.RawURLEncoding.EncodeToString([]byte(`{"x":"` + s + `"}`))
			}),
		).Draw(t, "bad_cursor")

		svc := &mockCredentialLogService{}
		r := setupCredentialLogRouter(svc, uuid.New().String())

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
			"/credentials/logs?cursor="+badCursor, http.NoBody)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("cursor=%q: expected 400, got %d (body=%s)", badCursor, w.Code, w.Body.String())
		}
	})
}

// ── Property 11: Response fields completeness ─────────────────────────────────

// TestProperty_CredentialLog_ResponseFields verifies that both list and GetByID
// responses contain all required fields and never expose key_hash or raw_key.
// Feature: credential-logs, Property 11: Response fields completeness
func TestProperty_CredentialLog_ResponseFields(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgID := uuid.New()
		logID := uuid.New()
		credID := uuid.New()
		now := time.Now().UTC().Truncate(time.Second)

		hasCredentialID := rapid.Bool().Draw(t, "has_credential_id")
		var credPtr *uuid.UUID
		if hasCredentialID {
			credPtr = &credID
		}

		record := domain.CredentialLogRecord{
			ID:             logID,
			OrganizationID: orgID,
			CredentialID:   credPtr,
			CreatedAt:      now,
		}

		svc := &mockCredentialLogService{
			listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, *string, error) {
				return []domain.CredentialLogRecord{record}, nil, nil
			},
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialLogRecord, error) {
				return &record, nil
			},
		}
		r := setupCredentialLogRouter(svc, orgID.String())

		requiredFields := []string{"id", "organization_id", "credential_id", "method", "path",
			"request_body", "response_body", "status_code", "latency_ms", "ip_address", "user_agent", "created_at"}
		forbiddenFields := []string{"key_hash", "raw_key"}

		// Check list response.
		wList := httptest.NewRecorder()
		reqList := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/credentials/logs", http.NoBody)
		r.ServeHTTP(wList, reqList)

		if wList.Code != http.StatusOK {
			t.Fatalf("list: expected 200, got %d (body=%s)", wList.Code, wList.Body.String())
		}
		var listResp map[string]any
		if err := json.Unmarshal(wList.Body.Bytes(), &listResp); err != nil {
			t.Fatalf("list: failed to parse response: %v", err)
		}
		data, _ := listResp["data"].([]any)
		if len(data) != 1 {
			t.Fatalf("list: expected 1 item, got %d", len(data))
		}
		item := data[0].(map[string]any)
		for _, field := range requiredFields {
			if _, ok := item[field]; !ok {
				t.Fatalf("list item missing required field %q", field)
			}
		}
		for _, field := range forbiddenFields {
			if _, ok := item[field]; ok {
				t.Fatalf("list item exposes forbidden field %q", field)
			}
		}

		// Check GetByID response.
		wGet := httptest.NewRecorder()
		reqGet := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
			"/credentials/logs/"+logID.String(), http.NoBody)
		r.ServeHTTP(wGet, reqGet)

		if wGet.Code != http.StatusOK {
			t.Fatalf("get: expected 200, got %d (body=%s)", wGet.Code, wGet.Body.String())
		}
		var getResp map[string]any
		if err := json.Unmarshal(wGet.Body.Bytes(), &getResp); err != nil {
			t.Fatalf("get: failed to parse response: %v", err)
		}
		for _, field := range requiredFields {
			if _, ok := getResp[field]; !ok {
				t.Fatalf("get response missing required field %q", field)
			}
		}
		for _, field := range forbiddenFields {
			if _, ok := getResp[field]; ok {
				t.Fatalf("get response exposes forbidden field %q", field)
			}
		}
	})
}

// ── Property 12: RBAC enforcement ────────────────────────────────────────────

// TestProperty_CredentialLog_RBACEnforcement verifies that member role gets 403
// and admin/owner roles do not get 403.
// Feature: credential-logs, Property 12: RBAC enforcement
func TestProperty_CredentialLog_RBACEnforcement(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		role := rapid.SampledFrom([]string{"member", "admin", "owner"}).Draw(t, "role")
		orgID := uuid.New()
		logID := uuid.New()

		svc := &mockCredentialLogService{
			listFn: func(_ context.Context, _ uuid.UUID, _ domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, *string, error) {
				return nil, nil, nil
			},
			getByIDFn: func(_ context.Context, _, _ uuid.UUID) (*domain.CredentialLogRecord, error) {
				return &domain.CredentialLogRecord{ID: logID, OrganizationID: orgID, CreatedAt: time.Now()}, nil
			},
		}
		r := setupCredentialLogRouterWithRole(svc, orgID.String(), role)

		endpoints := []struct {
			method string
			path   string
		}{
			{http.MethodGet, "/credentials/logs"},
			{http.MethodGet, "/credentials/logs/" + logID.String()},
		}

		for _, ep := range endpoints {
			w := httptest.NewRecorder()
			req := httptest.NewRequestWithContext(context.Background(), ep.method, ep.path, http.NoBody)
			r.ServeHTTP(w, req)

			switch role {
			case "member":
				if w.Code != http.StatusForbidden {
					t.Fatalf("member: %s %s expected 403, got %d", ep.method, ep.path, w.Code)
				}
			case "admin", "owner":
				if w.Code == http.StatusForbidden {
					t.Fatalf("%s: %s %s expected non-403, got %d", role, ep.method, ep.path, w.Code)
				}
			}
		}
	})
}

// ── Property 13: Invalid filter params return 400 ────────────────────────────

// TestProperty_CredentialLog_InvalidFilterParams verifies that invalid from/to
// timestamps, non-integer status_code, and from > to all return 400.
// Feature: credential-logs, Property 13: Invalid filter params return 400
func TestProperty_CredentialLog_InvalidFilterParams(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		svc := &mockCredentialLogService{}
		r := setupCredentialLogRouter(svc, uuid.New().String())

		// Pick one of the invalid param scenarios.
		scenario := rapid.IntRange(0, 3).Draw(t, "scenario")
		var query string
		switch scenario {
		case 0:
			// Invalid from timestamp.
			query = "from=not-a-date"
		case 1:
			// Invalid to timestamp.
			query = "to=not-a-date"
		case 2:
			// Non-integer status_code.
			query = "status_code=abc"
		case 3:
			// from > to (valid timestamps but wrong order).
			query = "from=2026-06-01T00:00:00Z&to=2026-01-01T00:00:00Z"
		}

		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodGet,
			"/credentials/logs?"+query, http.NoBody)
		r.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Fatalf("query=%q: expected 400, got %d (body=%s)", query, w.Code, w.Body.String())
		}
	})
}
