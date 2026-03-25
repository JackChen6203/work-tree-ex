package main

import (
	"context"
	"os"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/budget"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/notifications"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/httpserver"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/logging"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/trips"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Environment)
	ctx := context.Background()

	if cfg.TripsStore == "postgres" {
		pool, err := database.Connect(ctx, cfg.Database)
		if err != nil {
			logger.Error("failed to connect postgres for trips repository", "error", err)
			os.Exit(1)
		}
		trips.SetRepository(trips.NewPostgresRepository(pool))
		trips.SetCollaborationPool(pool)
		notifications.SetPool(pool)
		budget.SetPool(pool)
		logger.Info("trips repository configured", "store", "postgres")
	} else {
		trips.SetCollaborationPool(nil)
		notifications.SetPool(nil)
		budget.SetPool(nil)
		logger.Info("trips repository configured", "store", "memory")
	}

	srv := httpserver.New(cfg, logger)
	if err := srv.Run(ctx); err != nil {
		logger.Error("api server stopped with error", "error", err)
		os.Exit(1)
	}
}
