package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/notifications"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/logging"
	syncmod "github.com/solidityDeveloper/time_tree_ex/backend/internal/sync"
)

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Environment)

	pollInterval := time.Duration(getEnvInt("WORKER_POLL_INTERVAL_SEC", 1)) * time.Second
	batchSize := getEnvInt("WORKER_BATCH_SIZE", 50)

	logger.Info("worker started",
		slog.String("store", cfg.TripsStore),
		slog.Duration("pollInterval", pollInterval),
		slog.Int("batchSize", batchSize),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if strings.EqualFold(strings.TrimSpace(cfg.TripsStore), "postgres") {
		pool, err := database.Connect(ctx, cfg.Database)
		if err != nil {
			logger.Error("worker failed to connect database", slog.String("error", err.Error()))
			os.Exit(1)
		}
		defer pool.Close()

		syncmod.SetPool(pool)
		notifications.SetPool(pool)
	} else {
		logger.Warn("worker running without postgres store; outbox polling disabled")
	}

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	defer signal.Stop(sigCh)

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		select {
		case sig := <-sigCh:
			logger.Info("worker shutdown signal received; exiting after current cycle", slog.String("signal", sig.String()))
			logger.Info("worker stopped")
			return
		case <-ticker.C:
			if !strings.EqualFold(strings.TrimSpace(cfg.TripsStore), "postgres") {
				continue
			}
			runOutboxBatch(ctx, logger, batchSize)
		}
	}
}

func runOutboxBatch(ctx context.Context, logger *slog.Logger, batchSize int) {
	events, err := syncmod.PollPendingOutboxEvents(ctx, batchSize)
	if err != nil {
		logger.Error("worker failed to poll outbox events", slog.String("error", err.Error()))
		return
	}
	if len(events) == 0 {
		return
	}

	logger.Info("worker polled outbox events", slog.Int("count", len(events)))
	for _, evt := range events {
		processErr := notifications.ConsumeOutboxEvent(ctx, evt.EventType, evt.TripID, evt.Payload)
		if processErr != nil {
			if _, ackErr := syncmod.AckOutboxEvent(ctx, evt.ID, false); ackErr != nil {
				logger.Error("worker failed to nack outbox event",
					slog.String("eventId", evt.ID),
					slog.String("processError", processErr.Error()),
					slog.String("ackError", ackErr.Error()),
				)
			} else {
				logger.Warn("worker processed outbox event with error",
					slog.String("eventId", evt.ID),
					slog.String("error", processErr.Error()),
				)
			}
			continue
		}

		if _, ackErr := syncmod.AckOutboxEvent(ctx, evt.ID, true); ackErr != nil {
			logger.Error("worker failed to ack outbox event", slog.String("eventId", evt.ID), slog.String("error", ackErr.Error()))
			continue
		}

		logger.Info("worker processed outbox event",
			slog.String("eventId", evt.ID),
			slog.String("eventType", evt.EventType),
			slog.String("aggregateType", evt.AggregateType),
			slog.String("aggregateId", evt.AggregateID),
		)
		// Placeholder analytics dispatch hook.
		logger.Info("worker emitted analytics event",
			slog.String("eventType", evt.EventType),
			slog.String("aggregateType", evt.AggregateType),
		)
	}
}

func getEnvInt(key string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return parsed
}
