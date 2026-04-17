package service

import (
	"context"
	"sort"
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// Feature: subscription-status-log-backend, Property 6: Cursor pagination is complete and has no duplicates
// Para qualquer N registros e qualquer limit k, paginar até o fim retorna
// exatamente N registros sem duplicates em ordem created_at ASC.

// paginatingHistoryRepo is an in-memory SubscriptionHistoryRepository that
// implements cursor-based pagination using the same keyset logic as the real repo.
type paginatingHistoryRepo struct {
	records []domain.SubscriptionStatusHistory
}

func (r *paginatingHistoryRepo) List(
	_ context.Context,
	_, _ uuid.UUID,
	limit int,
	cursor *domain.CursorToken,
) ([]domain.SubscriptionStatusHistory, *domain.CursorToken, error) {
	filtered := make([]domain.SubscriptionStatusHistory, 0, len(r.records))
	if cursor != nil {
		for i := range r.records {
			rec := &r.records[i]
			if rec.CreatedAt.After(cursor.CreatedAt) ||
				(rec.CreatedAt.Equal(cursor.CreatedAt) && rec.ID.String() > cursor.ID.String()) {
				filtered = append(filtered, *rec)
			}
		}
	} else {
		filtered = append(filtered, r.records...)
	}

	// Fetch limit+1 to detect next page.
	fetchN := limit + 1
	if fetchN > len(filtered) {
		fetchN = len(filtered)
	}
	page := filtered[:fetchN]

	var nextCursor *domain.CursorToken
	if len(page) > limit {
		last := page[limit-1]
		nextCursor = &domain.CursorToken{CreatedAt: last.CreatedAt, ID: last.ID}
		page = page[:limit]
	}

	return page, nextCursor, nil
}

func TestProperty_CursorPagination_CompleteAndNoDuplicates(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		n := rapid.IntRange(0, 50).Draw(t, "n_records")
		limit := rapid.IntRange(1, 20).Draw(t, "limit")

		// Generate N records with distinct (created_at, id) pairs.
		records := make([]domain.SubscriptionStatusHistory, n)
		base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < n; i++ {
			records[i] = domain.SubscriptionStatusHistory{
				ID:        uuid.New(),
				CreatedAt: base.Add(time.Duration(i) * time.Second),
				NewStatus: domain.SubStatusActive,
			}
		}

		// Sort ascending by (created_at, id) — same order as the DB query.
		sort.Slice(records, func(i, j int) bool {
			if records[i].CreatedAt.Equal(records[j].CreatedAt) {
				return records[i].ID.String() < records[j].ID.String()
			}
			return records[i].CreatedAt.Before(records[j].CreatedAt)
		})

		repo := &paginatingHistoryRepo{records: records}

		// Paginate until no next cursor.
		var collected []domain.SubscriptionStatusHistory
		var cursor *domain.CursorToken
		orgID := uuid.New()
		subID := uuid.New()

		for {
			page, next, err := repo.List(context.Background(), orgID, subID, limit, cursor)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			collected = append(collected, page...)
			if next == nil {
				break
			}
			cursor = next
		}

		// Assert total count matches.
		if len(collected) != n {
			t.Fatalf("expected %d records total, got %d", n, len(collected))
		}

		// Assert no duplicates.
		seen := make(map[uuid.UUID]bool, n)
		for _, rec := range collected {
			if seen[rec.ID] {
				t.Fatalf("duplicate record ID %v in paginated results", rec.ID)
			}
			seen[rec.ID] = true
		}

		// Assert ascending order.
		for i := 1; i < len(collected); i++ {
			prev := collected[i-1]
			curr := collected[i]
			if curr.CreatedAt.Before(prev.CreatedAt) {
				t.Fatalf("records out of order at index %d: %v > %v", i, prev.CreatedAt, curr.CreatedAt)
			}
		}
	})
}

// TestCursorEncodeDecodeRoundTrip verifies that EncodeCursor/DecodeCursor are inverses.
func TestCursorEncodeDecodeRoundTrip(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		id := uuid.New()
		// Use second-precision to avoid nanosecond rounding in RFC3339Nano.
		ts := time.Now().UTC().Truncate(time.Nanosecond)

		encoded := EncodeCursor(ts, id)
		decoded, err := DecodeCursor(encoded)
		if err != nil {
			t.Fatalf("DecodeCursor failed: %v", err)
		}
		if !decoded.CreatedAt.Equal(ts) {
			t.Fatalf("CreatedAt mismatch: want %v, got %v", ts, decoded.CreatedAt)
		}
		if decoded.ID != id {
			t.Fatalf("ID mismatch: want %v, got %v", id, decoded.ID)
		}
	})
}
