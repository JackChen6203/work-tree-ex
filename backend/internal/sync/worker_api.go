package sync

import (
	"context"
	"errors"
	"time"
)

type OutboxStats struct {
	PendingCount   int `json:"pendingCount"`
	ProcessedCount int `json:"processedCount"`
	DLQCount       int `json:"dlqCount"`
}

func PollPendingOutboxEvents(ctx context.Context, limit int) ([]OutboxEvent, error) {
	if getPool() != nil {
		items, err := listOutboxEventsPostgres(ctx, "pending")
		if err != nil {
			return nil, err
		}
		if limit > 0 && len(items) > limit {
			return items[:limit], nil
		}
		return items, nil
	}

	syncMu.RLock()
	defer syncMu.RUnlock()

	now := time.Now().UTC()
	items := make([]OutboxEvent, 0)
	for _, evt := range outboxEvents {
		if evt.Status != "pending" || evt.AvailableAt.After(now) {
			continue
		}
		items = append(items, evt)
		if limit > 0 && len(items) >= limit {
			break
		}
	}
	return items, nil
}

func AckOutboxEvent(ctx context.Context, eventID string, success bool) (OutboxEvent, error) {
	if getPool() != nil {
		return ackOutboxEventPostgres(ctx, eventID, success)
	}

	syncMu.Lock()
	defer syncMu.Unlock()

	evt, ok := outboxByID[eventID]
	if !ok {
		return OutboxEvent{}, ErrOutboxEventNotFound
	}

	if success {
		now := time.Now().UTC()
		evt.Status = "processed"
		evt.ProcessedAt = &now
		return *evt, nil
	}

	evt.RetryCount++
	if evt.RetryCount > maxRetries {
		evt.Status = "dlq"
		return *evt, nil
	}
	evt.AvailableAt = time.Now().UTC().Add(time.Duration(1<<evt.RetryCount) * time.Second)
	return *evt, nil
}

func RetryOutboxEvent(ctx context.Context, eventID string) (OutboxEvent, error) {
	if getPool() != nil {
		return retryOutboxEventPostgres(ctx, eventID)
	}

	syncMu.Lock()
	defer syncMu.Unlock()

	evt, ok := outboxByID[eventID]
	if !ok {
		return OutboxEvent{}, ErrOutboxEventNotFound
	}

	evt.Status = "pending"
	evt.RetryCount = 0
	evt.ProcessedAt = nil
	evt.AvailableAt = time.Now().UTC()
	return *evt, nil
}

func GetOutboxStats(ctx context.Context) (OutboxStats, error) {
	if getPool() != nil {
		return getOutboxStatsPostgres(ctx)
	}

	syncMu.RLock()
	defer syncMu.RUnlock()

	stats := OutboxStats{}
	for _, evt := range outboxEvents {
		switch evt.Status {
		case "pending":
			stats.PendingCount++
		case "processed":
			stats.ProcessedCount++
		case "dlq":
			stats.DLQCount++
		}
	}
	return stats, nil
}

func IsOutboxNotFound(err error) bool {
	return errors.Is(err, ErrOutboxEventNotFound)
}
