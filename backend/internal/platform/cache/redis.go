package cache

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
)

var (
	redisMu     sync.RWMutex
	redisClient redis.UniversalClient
)

func ConnectRedis(ctx context.Context, cfg config.RedisConfig) (redis.UniversalClient, error) {
	addr := strings.TrimSpace(cfg.Addr)
	if addr == "" {
		return nil, errors.New("redis addr is required")
	}

	opts := &redis.Options{
		Addr:            addr,
		Password:        cfg.Password,
		DB:              cfg.DB,
		PoolSize:        cfg.PoolSize,
		MinIdleConns:    cfg.MinIdleConns,
		ConnMaxLifetime: cfg.ConnMaxLifetime,
		ConnMaxIdleTime: cfg.ConnMaxIdleTime,
	}

	client := redis.NewClient(opts)
	pingCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := client.Ping(pingCtx).Err(); err != nil {
		_ = client.Close()
		return nil, err
	}

	return client, nil
}

func SetRedisClient(client redis.UniversalClient) {
	redisMu.Lock()
	defer redisMu.Unlock()
	redisClient = client
}

func GetRedisClient() redis.UniversalClient {
	redisMu.RLock()
	defer redisMu.RUnlock()
	return redisClient
}

func CloseRedisClient() error {
	redisMu.Lock()
	defer redisMu.Unlock()
	if redisClient == nil {
		return nil
	}
	err := redisClient.Close()
	redisClient = nil
	return err
}
