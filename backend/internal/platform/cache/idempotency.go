package cache

import (
	"context"
	"strings"
	"time"
)

const tripCreateIdempotencyTTL = 24 * time.Hour

func GetTripIDByIdempotencyKey(ctx context.Context, idempotencyKey string) (string, bool) {
	if !DistributedModeEnabled() {
		return "", false
	}
	client := GetRedisClient()
	if client == nil {
		return "", false
	}

	key := strings.TrimSpace(idempotencyKey)
	if key == "" {
		return "", false
	}

	value, err := client.Get(ctx, tripCreateIdempotencyRedisKey(key)).Result()
	if err != nil || strings.TrimSpace(value) == "" {
		return "", false
	}
	return value, true
}

func SetTripIDByIdempotencyKey(ctx context.Context, idempotencyKey, tripID string) {
	if !DistributedModeEnabled() {
		return
	}
	client := GetRedisClient()
	if client == nil {
		return
	}

	key := strings.TrimSpace(idempotencyKey)
	value := strings.TrimSpace(tripID)
	if key == "" || value == "" {
		return
	}
	_ = client.Set(ctx, tripCreateIdempotencyRedisKey(key), value, tripCreateIdempotencyTTL).Err()
}

func tripCreateIdempotencyRedisKey(idempotencyKey string) string {
	return "idempotency:trip:create:" + idempotencyKey
}
