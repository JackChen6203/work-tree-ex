package main

import (
	"context"
	"os"
	"time"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/admin"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/ai"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/auth"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/budget"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/itinerary"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/notifications"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/cache"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/httpserver"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/logging"
	syncpkg "github.com/solidityDeveloper/time_tree_ex/backend/internal/sync"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/trips"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/users"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Environment)
	ctx := context.Background()
	cache.SetDistributedMode(cfg.DistributedModeEnabled())
	if cache.DistributedModeEnabled() {
		if redisClient, err := cache.ConnectRedis(ctx, cfg.Redis); err != nil {
			logger.Warn("redis is unavailable, distributed features will fallback to single-host behavior", "addr", cfg.Redis.Addr, "error", err)
			cache.SetRedisClient(nil)
		} else {
			cache.SetRedisClient(redisClient)
			defer func() {
				if closeErr := cache.CloseRedisClient(); closeErr != nil {
					logger.Warn("failed to close redis client", "error", closeErr)
				}
			}()
			logger.Info("distributed runtime mode enabled", "redis_addr", cfg.Redis.Addr)
		}
	} else {
		cache.SetRedisClient(nil)
		logger.Info("single-host runtime mode enabled; distributed redis features are disabled")
	}

	if cfg.TripsStore == "postgres" {
		if err := database.RunMigrations(ctx, cfg.Database); err != nil {
			logger.Error("failed to run database migrations", "error", err)
			os.Exit(1)
		}
		pool, err := database.Connect(ctx, cfg.Database)
		if err != nil {
			logger.Error("failed to connect postgres for trips repository", "error", err)
			os.Exit(1)
		}
		trips.SetRepository(trips.NewPostgresRepository(pool))
		trips.SetCollaborationPool(pool)
		notifications.SetPool(pool)
		budget.SetPool(pool)
		itinerary.SetPool(pool)
		auth.SetPool(pool)
		admin.SetPool(pool)
		ai.SetPool(pool)
		syncpkg.SetPool(pool)
		users.SetPool(pool)
		auth.StartSessionCleanupWorker(ctx, time.Hour)
		notifications.StartFCMTokenCleanupWorker(ctx, 6*time.Hour, 30*24*time.Hour)
		httpserver.SetReadinessProbe(func(ctx context.Context) error {
			return pool.Ping(ctx)
		})
		logger.Info("trips repository configured", "store", "postgres")
	} else {
		trips.SetCollaborationPool(nil)
		notifications.SetPool(nil)
		budget.SetPool(nil)
		itinerary.SetPool(nil)
		auth.SetPool(nil)
		admin.SetPool(nil)
		ai.SetPool(nil)
		syncpkg.SetPool(nil)
		users.SetPool(nil)
		httpserver.SetReadinessProbe(nil)
		logger.Info("trips repository configured", "store", "memory")
	}

	srv := httpserver.New(cfg, logger)
	if err := srv.Run(ctx); err != nil {
		logger.Error("api server stopped with error", "error", err)
		os.Exit(1)
	}
}
