package ai

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/cache"
)

const tripPlanningLockTTL = 45 * time.Second

var releaseTripPlanningLockScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("DEL", KEYS[1])
end
return 0
`)

func acquireTripPlanningLock(ctx context.Context, tripID string) (release func(), acquired bool) {
	if cache.DistributedModeEnabled() {
		if client := cache.GetRedisClient(); client != nil {
			lockKey := "ai:plan-lock:" + tripID
			lockToken := uuid.NewString()
			if ok, err := client.SetNX(ctx, lockKey, lockToken, tripPlanningLockTTL).Result(); err == nil {
				if !ok {
					return nil, false
				}
				return func() {
					unlockCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
					defer cancel()
					_, _ = releaseTripPlanningLockScript.Run(unlockCtx, client, []string{lockKey}, lockToken).Result()
				}, true
			}
		}
	}

	plannerMu.Lock()
	defer plannerMu.Unlock()
	if tripJobLocks[tripID] {
		return nil, false
	}
	tripJobLocks[tripID] = true
	return func() {
		plannerMu.Lock()
		delete(tripJobLocks, tripID)
		plannerMu.Unlock()
	}, true
}
