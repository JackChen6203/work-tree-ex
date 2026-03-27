package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/solidityDeveloper/time_tree_ex/backend/internal/notifications"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/config"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
	"github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/logging"
	syncmod "github.com/solidityDeveloper/time_tree_ex/backend/internal/sync"
)

type workerRuntimeMetrics struct {
	startedAt      time.Time
	lastPollAtUnix atomic.Int64
	batchCount     atomic.Int64
	polledCount    atomic.Int64
	processedCount atomic.Int64
	failedCount    atomic.Int64
	dlqCount       atomic.Int64
}

func newWorkerRuntimeMetrics() *workerRuntimeMetrics {
	m := &workerRuntimeMetrics{
		startedAt: time.Now().UTC(),
	}
	m.lastPollAtUnix.Store(m.startedAt.Unix())
	return m
}

func (m *workerRuntimeMetrics) markBatch(events int) {
	if m == nil {
		return
	}
	m.batchCount.Add(1)
	m.polledCount.Add(int64(events))
	m.lastPollAtUnix.Store(time.Now().UTC().Unix())
}

func (m *workerRuntimeMetrics) markProcessed() {
	if m == nil {
		return
	}
	m.processedCount.Add(1)
}

func (m *workerRuntimeMetrics) markFailed() {
	if m == nil {
		return
	}
	m.failedCount.Add(1)
}

func (m *workerRuntimeMetrics) markDLQ() {
	if m == nil {
		return
	}
	m.dlqCount.Add(1)
}

func main() {
	cfg := config.Load()
	logger := logging.New(cfg.Environment)

	pollInterval := time.Duration(getEnvInt("WORKER_POLL_INTERVAL_SEC", 1)) * time.Second
	batchSize := getEnvInt("WORKER_BATCH_SIZE", 50)
	metrics := newWorkerRuntimeMetrics()

	logger.Info("worker started",
		slog.String("store", cfg.TripsStore),
		slog.Duration("pollInterval", pollInterval),
		slog.Int("batchSize", batchSize),
	)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	startWorkerMonitorServer(ctx, logger, metrics, pollInterval)

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
			runOutboxBatch(ctx, logger, batchSize, metrics)
		}
	}
}

func runOutboxBatch(ctx context.Context, logger *slog.Logger, batchSize int, metrics *workerRuntimeMetrics) {
	events, err := syncmod.PollPendingOutboxEvents(ctx, batchSize)
	if err != nil {
		logger.Error("worker failed to poll outbox events", slog.String("error", err.Error()))
		return
	}
	metrics.markBatch(len(events))
	if len(events) == 0 {
		return
	}

	logger.Info("worker polled outbox events", slog.Int("count", len(events)))
	for _, evt := range events {
		processErr := notifications.ConsumeOutboxEvent(ctx, evt.EventType, evt.TripID, evt.Payload)
		if processErr != nil {
			metrics.markFailed()
			updated, ackErr := syncmod.AckOutboxEvent(ctx, evt.ID, false)
			if ackErr != nil {
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
				if updated.Status == "dlq" {
					metrics.markDLQ()
					logger.Error("worker moved outbox event to dlq",
						slog.String("eventId", evt.ID),
						slog.String("eventType", evt.EventType),
					)
				}
			}
			continue
		}

		if _, ackErr := syncmod.AckOutboxEvent(ctx, evt.ID, true); ackErr != nil {
			logger.Error("worker failed to ack outbox event", slog.String("eventId", evt.ID), slog.String("error", ackErr.Error()))
			continue
		}
		metrics.markProcessed()

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

func startWorkerMonitorServer(ctx context.Context, logger *slog.Logger, metrics *workerRuntimeMetrics, pollInterval time.Duration) {
	port := getEnvInt("WORKER_HTTP_PORT", 8091)
	if port <= 0 {
		logger.Info("worker monitor server disabled", slog.Int("port", port))
		return
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status": "ok",
		})
	})
	mux.HandleFunc("/readyz", func(w http.ResponseWriter, _ *http.Request) {
		lastPoll := time.Unix(metrics.lastPollAtUnix.Load(), 0).UTC()
		staleAfter := 3 * pollInterval
		if staleAfter < 3*time.Second {
			staleAfter = 3 * time.Second
		}
		if time.Since(lastPoll) > staleAfter {
			writeJSON(w, http.StatusServiceUnavailable, map[string]any{
				"status":   "not_ready",
				"lastPoll": lastPoll,
			})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"status":   "ready",
			"lastPoll": lastPoll,
		})
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		outboxStats, err := syncmod.GetOutboxStats(r.Context())
		payload := map[string]any{
			"startedAt": metrics.startedAt,
			"polling": map[string]any{
				"lastPollAt": time.Unix(metrics.lastPollAtUnix.Load(), 0).UTC(),
				"batchCount": metrics.batchCount.Load(),
			},
			"events": map[string]any{
				"polled":    metrics.polledCount.Load(),
				"processed": metrics.processedCount.Load(),
				"failed":    metrics.failedCount.Load(),
				"dlqMoved":  metrics.dlqCount.Load(),
			},
			"outbox": outboxStats,
		}
		if err != nil {
			payload["outboxError"] = err.Error()
		}
		writeJSON(w, http.StatusOK, payload)
	})

	server := &http.Server{
		Addr:              ":" + strconv.Itoa(port),
		Handler:           mux,
		ReadHeaderTimeout: 2 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("worker monitor server stopped with error", slog.String("error", err.Error()))
		}
	}()

	logger.Info("worker monitor server started", slog.Int("port", port))
}

func writeJSON(w http.ResponseWriter, status int, payload map[string]any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
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
