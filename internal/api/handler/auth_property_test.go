package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"payflow/internal/domain"
	"payflow/internal/service"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// TestProperty_AuthHandler_RandomAPIKeyAlwaysRejects verifies that random api_key values always return 401.
func TestProperty_AuthHandler_RandomAPIKeyAlwaysRejects(t *testing.T) {
	svc := &mockAuthService{
		loginFn: func(_ context.Context, _ domain.LoginRequest) (*service.LoginResult, error) {
			return nil, domain.ErrUnauthorized
		},
	}
	r := setupAuthRouter(svc)

	rapid.Check(t, func(t *rapid.T) {
		apiKey := rapid.String().Draw(t, "api_key")

		body, _ := json.Marshal(map[string]string{"api_key": apiKey})
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/login", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code != http.StatusUnauthorized && w.Code != http.StatusBadRequest {
			t.Fatalf("api_key=%q: expected 401 or 400, got %d (body=%s)", apiKey, w.Code, w.Body.String())
		}
	})
}

// TestProperty_AuthHandler_SelectOrgRandomInputsNever200or500 verifies Property 7:
// random (session_token, organization_id) inputs never return 200 or 500.
// Feature: critical-tests-coverage, Property 7: select-org random inputs never 200 or 500
func TestProperty_AuthHandler_SelectOrgRandomInputsNever200or500(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		sessionToken := rapid.String().Draw(t, "session_token")
		orgID := uuid.New().String()

		// Mock always returns ErrInvalidSessionToken for any input
		svc := &mockAuthService{}
		r := setupAuthRouter(svc)

		body, _ := json.Marshal(map[string]string{
			"session_token":   sessionToken,
			"organization_id": orgID,
		})
		w := httptest.NewRecorder()
		req := httptest.NewRequestWithContext(context.Background(), http.MethodPost, "/auth/select-organization", strings.NewReader(string(body)))
		req.Header.Set("Content-Type", "application/json")
		r.ServeHTTP(w, req)

		if w.Code == http.StatusOK || w.Code == http.StatusInternalServerError {
			t.Fatalf("session_token=%q: expected 400 or 401, got %d (body=%s)", sessionToken, w.Code, w.Body.String())
		}
	})
}
