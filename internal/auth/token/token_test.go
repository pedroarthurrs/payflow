package token_test

import (
	"testing"
	"time"

	"i9pay/internal/auth/token"
	"i9pay/internal/domain"

	"pgregory.net/rapid"
)

// TestProperty_ClaimsOrganizationIDAndAuthType verifies that for any authentication mode
// (user or system), the emitted token always contains a non-empty organization_id and
// a valid auth_type ("user" or "system").
func TestProperty_ClaimsOrganizationIDAndAuthType(t *testing.T) {
	svc := token.NewService("test-secret-key", 15*time.Minute, 7*24*time.Hour)

	validAuthTypes := []domain.AuthType{domain.AuthTypeUser, domain.AuthTypeSystem}

	rapid.Check(t, func(t *rapid.T) {
		organizationID := rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "organizationID")
		authType := rapid.SampledFrom(validAuthTypes).Draw(t, "authType")

		var userID string
		var role domain.Role
		if authType == domain.AuthTypeUser {
			userID = rapid.StringMatching(`[a-f0-9]{8}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{4}-[a-f0-9]{12}`).Draw(t, "userID")
			role = rapid.SampledFrom([]domain.Role{domain.RoleOwner, domain.RoleAdmin, domain.RoleDeveloper, domain.RoleSupport}).Draw(t, "role")
		}

		input := token.ClaimsInput{
			OrganizationID: organizationID,
			UserID:         userID,
			Role:           role,
			AuthType:       authType,
		}

		pair, err := svc.NewPairWithClaims(input)
		if err != nil {
			t.Fatalf("NewPairWithClaims failed: %v", err)
		}

		// Verify access token claims
		claims, err := svc.Verify(pair.AccessToken, token.TypeAccess)
		if err != nil {
			t.Fatalf("Verify access token failed: %v", err)
		}

		if claims.OrganizationID == "" {
			t.Fatal("access token: organization_id must not be empty")
		}
		if claims.OrganizationID != organizationID {
			t.Fatalf("access token: organization_id mismatch: got %q, want %q", claims.OrganizationID, organizationID)
		}
		if claims.AuthType != authType {
			t.Fatalf("access token: auth_type mismatch: got %q, want %q", claims.AuthType, authType)
		}
		if claims.AuthType != domain.AuthTypeUser && claims.AuthType != domain.AuthTypeSystem {
			t.Fatalf("access token: auth_type %q is not a valid value", claims.AuthType)
		}

		// Verify refresh token claims
		refreshClaims, err := svc.Verify(pair.RefreshToken, token.TypeRefresh)
		if err != nil {
			t.Fatalf("Verify refresh token failed: %v", err)
		}

		if refreshClaims.OrganizationID == "" {
			t.Fatal("refresh token: organization_id must not be empty")
		}
		if refreshClaims.OrganizationID != organizationID {
			t.Fatalf("refresh token: organization_id mismatch: got %q, want %q", refreshClaims.OrganizationID, organizationID)
		}
		if refreshClaims.AuthType != authType {
			t.Fatalf("refresh token: auth_type mismatch: got %q, want %q", refreshClaims.AuthType, authType)
		}
		if refreshClaims.AuthType != domain.AuthTypeUser && refreshClaims.AuthType != domain.AuthTypeSystem {
			t.Fatalf("refresh token: auth_type %q is not a valid value", refreshClaims.AuthType)
		}
	})
}
