package ai

import (
	"context"
	"testing"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/cache"
)

func TestAcquireTripPlanningLockFallsBackToLocalMutex(t *testing.T) {
	cache.SetDistributedMode(false)
	cache.SetRedisClient(nil)

	plannerMu.Lock()
	tripJobLocks = map[string]bool{}
	plannerMu.Unlock()

	release, acquired := acquireTripPlanningLock(context.Background(), "trip-local-1")
	if !acquired {
		t.Fatalf("expected first lock acquire to succeed")
	}
	if release == nil {
		t.Fatalf("expected release function")
	}

	release2, acquired2 := acquireTripPlanningLock(context.Background(), "trip-local-1")
	if acquired2 {
		t.Fatalf("expected second lock acquire to fail for same trip")
	}
	if release2 != nil {
		t.Fatalf("expected nil release when lock is not acquired")
	}

	release()

	release3, acquired3 := acquireTripPlanningLock(context.Background(), "trip-local-1")
	if !acquired3 {
		t.Fatalf("expected lock to be acquirable after release")
	}
	if release3 == nil {
		t.Fatalf("expected release function after re-acquire")
	}
	release3()
}
