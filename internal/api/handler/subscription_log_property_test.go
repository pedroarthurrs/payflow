package handler

import (
	"testing"
	"time"

	"i9pay/internal/domain"

	"github.com/google/uuid"
	"pgregory.net/rapid"
)

// Feature: subscription-status-log-backend, Property 5: Mapeamento handler preserva todos os campos do domínio
// Para qualquer SubscriptionStatusHistory gerado pelo rapid,
// toHistoryResponse(h) preserva todos os campos: id, old_status, new_status,
// event_source, event_source_id, reason, created_at.
func TestProperty_HistoryResponseMapping_PreservesAllFields(t *testing.T) {
	allStatuses := []domain.SubscriptionStatus{
		domain.SubStatusTrialing,
		domain.SubStatusActive,
		domain.SubStatusPastDue,
		domain.SubStatusUnpaid,
		domain.SubStatusCanceled,
		domain.SubStatusPaused,
		domain.SubStatusIncomplete,
	}

	rapid.Check(t, func(t *rapid.T) {
		id := uuid.New()
		orgID := uuid.New()
		subID := uuid.New()
		newStatusIdx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "new_status_idx")
		newStatus := allStatuses[newStatusIdx]
		eventSource := rapid.StringMatching(`[a-z]{3,10}`).Draw(t, "event_source")
		reason := rapid.StringMatching(`[a-z_]{3,20}`).Draw(t, "reason")
		createdAt := time.Now().UTC().Truncate(time.Second)

		// Optionally set old_status and event_source_id.
		var oldStatus *domain.SubscriptionStatus
		if rapid.Bool().Draw(t, "has_old_status") {
			idx := rapid.IntRange(0, len(allStatuses)-1).Draw(t, "old_status_idx")
			s := allStatuses[idx]
			oldStatus = &s
		}
		var eventSourceID *uuid.UUID
		if rapid.Bool().Draw(t, "has_event_source_id") {
			id := uuid.New()
			eventSourceID = &id
		}

		h := domain.SubscriptionStatusHistory{
			ID:             id,
			OrganizationID: orgID,
			SubscriptionID: subID,
			OldStatus:      oldStatus,
			NewStatus:      newStatus,
			EventSource:    eventSource,
			EventSourceID:  eventSourceID,
			Reason:         reason,
			CreatedAt:      createdAt,
		}

		resp := toHistoryResponse(h)

		if resp.ID != id {
			t.Fatalf("ID mismatch: want %v, got %v", id, resp.ID)
		}
		if resp.NewStatus != string(newStatus) {
			t.Fatalf("NewStatus mismatch: want %q, got %q", newStatus, resp.NewStatus)
		}
		if resp.EventSource != eventSource {
			t.Fatalf("EventSource mismatch: want %q, got %q", eventSource, resp.EventSource)
		}
		if resp.Reason != reason {
			t.Fatalf("Reason mismatch: want %q, got %q", reason, resp.Reason)
		}
		if !resp.CreatedAt.Equal(createdAt) {
			t.Fatalf("CreatedAt mismatch: want %v, got %v", createdAt, resp.CreatedAt)
		}

		// OldStatus
		if oldStatus == nil && resp.OldStatus != nil {
			t.Fatalf("expected nil OldStatus, got %q", *resp.OldStatus)
		}
		if oldStatus != nil {
			if resp.OldStatus == nil {
				t.Fatalf("expected OldStatus=%q, got nil", *oldStatus)
			}
			if *resp.OldStatus != string(*oldStatus) {
				t.Fatalf("OldStatus mismatch: want %q, got %q", *oldStatus, *resp.OldStatus)
			}
		}

		// EventSourceID
		if eventSourceID == nil && resp.EventSourceID != nil {
			t.Fatalf("expected nil EventSourceID, got %v", resp.EventSourceID)
		}
		if eventSourceID != nil {
			if resp.EventSourceID == nil {
				t.Fatalf("expected EventSourceID=%v, got nil", *eventSourceID)
			}
			if *resp.EventSourceID != *eventSourceID {
				t.Fatalf("EventSourceID mismatch: want %v, got %v", *eventSourceID, *resp.EventSourceID)
			}
		}
	})
}
