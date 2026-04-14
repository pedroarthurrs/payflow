package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// ── Property 1: Cursor round-trip ─────────────────────────────────────────────

// TestProperty_CredentialLog_CursorRoundTrip verifies that for any valid
// (created_at, id) pair, encoding then decoding produces the original values.
// Feature: credential-logs, Property 1: Cursor round-trip
func TestProperty_CredentialLog_CursorRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		// Generate a random time within a reasonable range.
		offsetSec := rapid.Int64Range(-1e9, 1e9).Draw(t, "offset_sec")
		ts := time.Now().UTC().Add(time.Duration(offsetSec) * time.Second).Truncate(time.Nanosecond)
		id := uuid.New()

		encoded := EncodeCursor(ts, id)
		decoded, err := DecodeCursor(encoded)
		if err != nil {
			t.Fatalf("unexpected decode error: %v", err)
		}
		if !decoded.CreatedAt.Equal(ts) {
			t.Fatalf("timestamp mismatch: got %v, want %v", decoded.CreatedAt, ts)
		}
		if decoded.ID != id {
			t.Fatalf("id mismatch: got %v, want %v", decoded.ID, id)
		}
	})
}

// ── Property 2: Pagination completeness ──────────────────────────────────────

// TestProperty_CredentialLog_PaginationCompleteness verifies that paginating
// through all records with a fixed limit yields exactly N unique IDs.
// Feature: credential-logs, Property 2: Pagination completeness
func TestProperty_CredentialLog_PaginationCompleteness(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 30).Draw(t, "n")
		limit := rapid.IntRange(1, 10).Draw(t, "limit")
		orgID := uuid.New()
		now := time.Now().UTC()

		// Build a sorted dataset (newest first).
		dataset := make([]domain.CredentialLogRecord, n)
		for i := range dataset {
			dataset[i] = domain.CredentialLogRecord{
				ID:        uuid.New(),
				CreatedAt: now.Add(-time.Duration(i) * time.Second),
			}
		}

		// Mock repo applies cursor filtering and limit in-memory.
		repo := &mockCredentialLogRepo{
			listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
				filtered := dataset
				if input.Cursor != nil {
					cutoff := input.Cursor
					var after []domain.CredentialLogRecord
					for _, r := range filtered {
						if r.CreatedAt.Before(cutoff.CreatedAt) ||
							(r.CreatedAt.Equal(cutoff.CreatedAt) && r.ID.String() < cutoff.ID.String()) {
							after = append(after, r)
						}
					}
					filtered = after
				}
				if input.Limit < len(filtered) {
					return filtered[:input.Limit], nil
				}
				return filtered, nil
			},
		}

		svc := NewCredentialService(nil, repo, nil, "test-pepper")
		seen := make(map[uuid.UUID]bool)
		var cursor *string

		for {
			input := domain.ListCredentialLogsInput{Limit: limit}
			if cursor != nil {
				ct, err := DecodeCursor(*cursor)
				if err != nil {
					t.Fatalf("failed to decode cursor: %v", err)
				}
				input.Cursor = ct
			}

			records, nextCursor, err := svc.ListLogs(context.Background(), orgID, input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			for _, r := range records {
				if seen[r.ID] {
					t.Fatalf("duplicate record ID: %v", r.ID)
				}
				seen[r.ID] = true
			}

			if nextCursor == nil {
				break
			}
			cursor = nextCursor
		}

		if len(seen) != n {
			t.Fatalf("expected %d unique records, got %d", n, len(seen))
		}
	})
}

// ── Property 3: Filter correctness — status_code ──────────────────────────────

// TestProperty_CredentialLog_FilterStatusCode verifies that all returned logs
// match the requested status_code filter.
// Feature: credential-logs, Property 3: Filter correctness — status_code
func TestProperty_CredentialLog_FilterStatusCode(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		targetSC := rapid.Int32Range(100, 599).Draw(t, "status_code")
		n := rapid.IntRange(1, 20).Draw(t, "n")
		orgID := uuid.New()
		now := time.Now().UTC()

		// Build dataset with mixed status codes.
		dataset := make([]domain.CredentialLogRecord, n)
		for i := range dataset {
			sc := rapid.Int32Range(100, 599).Draw(t, "sc")
			dataset[i] = domain.CredentialLogRecord{
				ID:         uuid.New(),
				StatusCode: &sc,
				CreatedAt:  now.Add(-time.Duration(i) * time.Second),
			}
		}

		repo := &mockCredentialLogRepo{
			listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
				var result []domain.CredentialLogRecord
				for _, r := range dataset {
					if input.StatusCode != nil && r.StatusCode != nil && *r.StatusCode == *input.StatusCode {
						result = append(result, r)
					}
				}
				if input.Limit < len(result) {
					return result[:input.Limit], nil
				}
				return result, nil
			},
		}

		svc := NewCredentialService(nil, repo, nil, "test-pepper")
		records, _, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{
			Limit:      100,
			StatusCode: &targetSC,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range records {
			if r.StatusCode == nil || *r.StatusCode != targetSC {
				t.Fatalf("record %v has status_code %v, expected %d", r.ID, r.StatusCode, targetSC)
			}
		}
	})
}

// ── Property 4: Filter correctness — method ───────────────────────────────────

// TestProperty_CredentialLog_FilterMethod verifies that all returned logs
// match the requested method filter.
// Feature: credential-logs, Property 4: Filter correctness — method
func TestProperty_CredentialLog_FilterMethod(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		methods := []string{"GET", "POST", "PUT", "PATCH", "DELETE"}
		targetMethod := rapid.SampledFrom(methods).Draw(t, "method")
		n := rapid.IntRange(1, 20).Draw(t, "n")
		orgID := uuid.New()
		now := time.Now().UTC()

		dataset := make([]domain.CredentialLogRecord, n)
		for i := range dataset {
			m := rapid.SampledFrom(methods).Draw(t, "m")
			dataset[i] = domain.CredentialLogRecord{
				ID:        uuid.New(),
				Method:    &m,
				CreatedAt: now.Add(-time.Duration(i) * time.Second),
			}
		}

		repo := &mockCredentialLogRepo{
			listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
				var result []domain.CredentialLogRecord
				for _, r := range dataset {
					if input.Method != nil && r.Method != nil && *r.Method == *input.Method {
						result = append(result, r)
					}
				}
				if input.Limit < len(result) {
					return result[:input.Limit], nil
				}
				return result, nil
			},
		}

		svc := NewCredentialService(nil, repo, nil, "test-pepper")
		records, _, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{
			Limit:  100,
			Method: &targetMethod,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range records {
			if r.Method == nil || *r.Method != targetMethod {
				t.Fatalf("record %v has method %v, expected %q", r.ID, r.Method, targetMethod)
			}
		}
	})
}

// ── Property 5: Filter correctness — path prefix ──────────────────────────────

// TestProperty_CredentialLog_FilterPathPrefix verifies that all returned logs
// have a path starting with the requested prefix.
// Feature: credential-logs, Property 5: Filter correctness — path prefix
func TestProperty_CredentialLog_FilterPathPrefix(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		prefix := rapid.StringMatching(`/[a-z]{1,10}`).Draw(t, "prefix")
		n := rapid.IntRange(1, 20).Draw(t, "n")
		orgID := uuid.New()
		now := time.Now().UTC()

		dataset := make([]domain.CredentialLogRecord, n)
		for i := range dataset {
			usePrefix := rapid.Bool().Draw(t, "use_prefix")
			var p string
			if usePrefix {
				p = prefix + "/resource"
			} else {
				p = "/other/path"
			}
			dataset[i] = domain.CredentialLogRecord{
				ID:        uuid.New(),
				Path:      &p,
				CreatedAt: now.Add(-time.Duration(i) * time.Second),
			}
		}

		repo := &mockCredentialLogRepo{
			listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
				var result []domain.CredentialLogRecord
				for _, r := range dataset {
					if input.PathPrefix != nil && r.Path != nil && strings.HasPrefix(*r.Path, *input.PathPrefix) {
						result = append(result, r)
					}
				}
				if input.Limit < len(result) {
					return result[:input.Limit], nil
				}
				return result, nil
			},
		}

		svc := NewCredentialService(nil, repo, nil, "test-pepper")
		records, _, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{
			Limit:      100,
			PathPrefix: &prefix,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range records {
			if r.Path == nil || !strings.HasPrefix(*r.Path, prefix) {
				t.Fatalf("record %v has path %v, expected prefix %q", r.ID, r.Path, prefix)
			}
		}
	})
}

// ── Property 6: Filter correctness — date range ───────────────────────────────

// TestProperty_CredentialLog_FilterDateRange verifies that all returned logs
// fall within the requested from/to date range.
// Feature: credential-logs, Property 6: Filter correctness — date range
func TestProperty_CredentialLog_FilterDateRange(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		base := time.Now().UTC().Truncate(time.Second)
		fromOffset := rapid.Int64Range(-3600, 0).Draw(t, "from_offset")
		toOffset := rapid.Int64Range(0, 3600).Draw(t, "to_offset")
		from := base.Add(time.Duration(fromOffset) * time.Second)
		to := base.Add(time.Duration(toOffset) * time.Second)
		n := rapid.IntRange(1, 20).Draw(t, "n")
		orgID := uuid.New()

		dataset := make([]domain.CredentialLogRecord, n)
		for i := range dataset {
			offsetSec := rapid.Int64Range(-7200, 7200).Draw(t, "ts_offset")
			dataset[i] = domain.CredentialLogRecord{
				ID:        uuid.New(),
				CreatedAt: base.Add(time.Duration(offsetSec) * time.Second),
			}
		}

		repo := &mockCredentialLogRepo{
			listFn: func(_ context.Context, _ uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
				var result []domain.CredentialLogRecord
				for _, r := range dataset {
					if input.From != nil && r.CreatedAt.Before(*input.From) {
						continue
					}
					if input.To != nil && r.CreatedAt.After(*input.To) {
						continue
					}
					result = append(result, r)
				}
				if input.Limit < len(result) {
					return result[:input.Limit], nil
				}
				return result, nil
			},
		}

		svc := NewCredentialService(nil, repo, nil, "test-pepper")
		records, _, err := svc.ListLogs(context.Background(), orgID, domain.ListCredentialLogsInput{
			Limit: 100,
			From:  &from,
			To:    &to,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, r := range records {
			if r.CreatedAt.Before(from) {
				t.Fatalf("record %v has created_at %v before from %v", r.ID, r.CreatedAt, from)
			}
			if r.CreatedAt.After(to) {
				t.Fatalf("record %v has created_at %v after to %v", r.ID, r.CreatedAt, to)
			}
		}
	})
}

// ── Property 7: Tenant isolation ─────────────────────────────────────────────

// TestProperty_CredentialLog_TenantIsolation verifies that logs for orgA never
// appear in orgB's List or GetByID responses.
// Feature: credential-logs, Property 7: Tenant isolation
func TestProperty_CredentialLog_TenantIsolation(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		orgA := uuid.New()
		orgB := uuid.New()
		for orgB == orgA {
			orgB = uuid.New()
		}
		now := time.Now().UTC()
		n := rapid.IntRange(1, 10).Draw(t, "n")

		logsA := make([]domain.CredentialLogRecord, n)
		for i := range logsA {
			logsA[i] = domain.CredentialLogRecord{
				ID:             uuid.New(),
				OrganizationID: orgA,
				CreatedAt:      now.Add(-time.Duration(i) * time.Second),
			}
		}

		repo := &mockCredentialLogRepo{
			listFn: func(_ context.Context, orgID uuid.UUID, input domain.ListCredentialLogsInput) ([]domain.CredentialLogRecord, error) {
				if orgID != orgA {
					return nil, nil
				}
				if input.Limit < len(logsA) {
					return logsA[:input.Limit], nil
				}
				return logsA, nil
			},
			getByIDFn: func(_ context.Context, orgID, logID uuid.UUID) (*domain.CredentialLogRecord, error) {
				if orgID == orgA {
					for _, r := range logsA {
						if r.ID == logID {
							return &r, nil
						}
					}
				}
				return nil, domain.ErrCredentialLogNotFound
			},
		}

		svc := NewCredentialService(nil, repo, nil, "test-pepper")

		// orgB list must be empty.
		records, _, err := svc.ListLogs(context.Background(), orgB, domain.ListCredentialLogsInput{Limit: 100})
		if err != nil {
			t.Fatalf("unexpected list error: %v", err)
		}
		if len(records) != 0 {
			t.Fatalf("expected 0 records for orgB, got %d", len(records))
		}

		// orgB GetByID for orgA's logs must return not-found.
		for _, r := range logsA {
			_, err := svc.GetLogByID(context.Background(), orgB, r.ID)
			if err != domain.ErrCredentialLogNotFound {
				t.Fatalf("expected ErrCredentialLogNotFound for orgB accessing orgA log %v, got %v", r.ID, err)
			}
		}
	})
}
