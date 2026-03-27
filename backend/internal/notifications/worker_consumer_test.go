package notifications

import (
	"context"
	"errors"
	"testing"
	"time"
)

type stubShadowSyncer struct {
	syncFn func(ctx context.Context, in shadowSyncInput) error
}

func (s stubShadowSyncer) Sync(ctx context.Context, in shadowSyncInput) error {
	if s.syncFn == nil {
		return nil
	}
	return s.syncFn(ctx, in)
}

func TestConsumeOutboxEventSyncsFirebaseShadow(t *testing.T) {
	SetPool(nil)
	resetPushGatewayForTest()
	resetShadowSyncerForTest()
	defer resetShadowSyncerForTest()

	notificationsMu.Lock()
	items = []notification{}
	pushDeliveries = map[string]*pushDelivery{}
	notificationsMu.Unlock()

	calls := 0
	setShadowSyncerForTest(stubShadowSyncer{
		syncFn: func(_ context.Context, in shadowSyncInput) error {
			calls++
			if in.EventType != "trip.updated" {
				t.Fatalf("unexpected event type: %s", in.EventType)
			}
			if in.ResourceID != "trip-1" {
				t.Fatalf("unexpected resource id: %s", in.ResourceID)
			}
			return nil
		},
	})

	err := ConsumeOutboxEvent(context.Background(), "trip.updated", "trip-1", map[string]any{
		"title": "Trip Updated",
		"body":  "Body",
		"link":  "/trips/trip-1",
	})
	if err != nil {
		t.Fatalf("consume outbox event failed: %v", err)
	}
	if calls != 1 {
		t.Fatalf("expected shadow sync called once, got %d", calls)
	}
}

func TestConsumeOutboxEventReturnsErrorWhenShadowSyncFails(t *testing.T) {
	SetPool(nil)
	resetPushGatewayForTest()
	resetShadowSyncerForTest()
	defer resetShadowSyncerForTest()

	notificationsMu.Lock()
	items = []notification{}
	pushDeliveries = map[string]*pushDelivery{}
	notificationsMu.Unlock()

	setShadowSyncerForTest(stubShadowSyncer{
		syncFn: func(_ context.Context, _ shadowSyncInput) error {
			return errors.New("shadow sync failed")
		},
	})

	err := ConsumeOutboxEvent(context.Background(), "trip.updated", "trip-2", map[string]any{
		"title": "Trip Updated",
		"body":  "Body",
		"link":  "/trips/trip-2",
	})
	if err == nil {
		t.Fatalf("expected shadow sync error")
	}
}

func TestPathSafeSegment(t *testing.T) {
	got := pathSafeSegment(" user/a:b ", "fallback")
	if got != "user_a_b" {
		t.Fatalf("unexpected safe segment: %s", got)
	}
	if pathSafeSegment("", "fallback") != "fallback" {
		t.Fatalf("expected fallback for empty value")
	}
}

func TestSyncFirebaseShadowNoopWhenDisabled(t *testing.T) {
	resetShadowSyncerForTest()
	defer resetShadowSyncerForTest()

	t.Setenv("FIREBASE_SHADOW_ENABLED", "false")
	if err := syncFirebaseShadow(context.Background(), "trip.updated", "trip-1", map[string]any{"k": "v"}, "u-1", "n-1"); err != nil {
		t.Fatalf("expected noop shadow sync when disabled, got %v", err)
	}
}

func TestSanitizeShadowKey(t *testing.T) {
	key := sanitizeShadowKey("n-1/abc:def")
	if key != "n-1_abc_def" {
		t.Fatalf("unexpected sanitized key: %s", key)
	}
	if sanitizeShadowKey("   ") != "" {
		t.Fatalf("expected empty key when input is blank")
	}
}

func TestRealtimeShadowSyncUsesNotificationIDAsKey(t *testing.T) {
	// This test only validates deterministic key generation behavior.
	now := time.Now().UTC()
	in := shadowSyncInput{
		NotificationID: "notif-123",
		Timestamp:      now,
	}
	key := sanitizeShadowKey(in.NotificationID)
	if key != "notif-123" {
		t.Fatalf("unexpected key: %s", key)
	}
}
