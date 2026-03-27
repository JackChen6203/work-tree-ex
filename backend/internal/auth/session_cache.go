package auth

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/cache"
)

const webSessionCachePrefix = "auth:web-session:"

func setCachedWebSession(ctx context.Context, sessionID string, user *sessionUser, ttl time.Duration) {
	if !cache.DistributedModeEnabled() || user == nil || ttl <= 0 {
		return
	}
	client := cache.GetRedisClient()
	if client == nil {
		return
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}

	payload, err := json.Marshal(user)
	if err != nil {
		return
	}

	_ = client.Set(ctx, webSessionCachePrefix+sessionID, payload, ttl).Err()
}

func getCachedWebSession(ctx context.Context, sessionID string) (*sessionUser, bool) {
	if !cache.DistributedModeEnabled() {
		return nil, false
	}
	client := cache.GetRedisClient()
	if client == nil {
		return nil, false
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return nil, false
	}

	raw, err := client.Get(ctx, webSessionCachePrefix+sessionID).Result()
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil, false
	}

	var user sessionUser
	if err := json.Unmarshal([]byte(raw), &user); err != nil {
		return nil, false
	}
	return &user, true
}

func deleteCachedWebSession(ctx context.Context, sessionID string) {
	if !cache.DistributedModeEnabled() {
		return
	}
	client := cache.GetRedisClient()
	if client == nil {
		return
	}

	sessionID = strings.TrimSpace(sessionID)
	if sessionID == "" {
		return
	}
	_ = client.Del(ctx, webSessionCachePrefix+sessionID).Err()
}
