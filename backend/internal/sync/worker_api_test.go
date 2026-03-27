package sync

import (
	"context"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func resetOutboxStateForWorkerTests() {
	SetPool(nil)
	syncMu.Lock()
	defer syncMu.Unlock()
	outboxEvents = []OutboxEvent{}
	outboxByID = map[string]*OutboxEvent{}
	outboxDedupeKeys = map[string]bool{}
}

func TestRetryOutboxEventResetsDLQEvent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	resetOutboxStateForWorkerTests()

	now := time.Now().UTC().Add(-2 * time.Minute)
	event := OutboxEvent{
		ID:          "evt-dlq-1",
		TripID:      "trip-1",
		EventType:   "itinerary_item.updated",
		Payload:     gin.H{"key": "value"},
		Status:      "dlq",
		RetryCount:  4,
		AvailableAt: now,
		CreatedAt:   now,
	}

	syncMu.Lock()
	outboxEvents = append(outboxEvents, event)
	outboxByID[event.ID] = &outboxEvents[0]
	syncMu.Unlock()

	updated, err := RetryOutboxEvent(context.Background(), event.ID)
	if err != nil {
		t.Fatalf("expected retry success, got error: %v", err)
	}
	if updated.Status != "pending" {
		t.Fatalf("expected pending status, got %s", updated.Status)
	}
	if updated.RetryCount != 0 {
		t.Fatalf("expected retryCount reset to 0, got %d", updated.RetryCount)
	}
	if updated.ProcessedAt != nil {
		t.Fatalf("expected processedAt cleared")
	}
}

func TestRetryOutboxEventNotFound(t *testing.T) {
	resetOutboxStateForWorkerTests()

	_, err := RetryOutboxEvent(context.Background(), "unknown-id")
	if !IsOutboxNotFound(err) {
		t.Fatalf("expected outbox not found error, got %v", err)
	}
}

func TestGetOutboxStatsMemory(t *testing.T) {
	resetOutboxStateForWorkerTests()

	now := time.Now().UTC()
	syncMu.Lock()
	outboxEvents = []OutboxEvent{
		{ID: "e1", Status: "pending", AvailableAt: now, CreatedAt: now},
		{ID: "e2", Status: "processed", AvailableAt: now, CreatedAt: now},
		{ID: "e3", Status: "dlq", AvailableAt: now, CreatedAt: now},
		{ID: "e4", Status: "pending", AvailableAt: now, CreatedAt: now},
	}
	syncMu.Unlock()

	stats, err := GetOutboxStats(context.Background())
	if err != nil {
		t.Fatalf("expected stats success, got %v", err)
	}
	if stats.PendingCount != 2 {
		t.Fatalf("expected pending=2, got %d", stats.PendingCount)
	}
	if stats.ProcessedCount != 1 {
		t.Fatalf("expected processed=1, got %d", stats.ProcessedCount)
	}
	if stats.DLQCount != 1 {
		t.Fatalf("expected dlq=1, got %d", stats.DLQCount)
	}
}
