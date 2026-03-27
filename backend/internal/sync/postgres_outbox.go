package sync

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformdb "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
)

var (
	ErrOutboxEventNotFound     = errors.New("outbox event not found")
	ErrOutboxEventNotRetryable = errors.New("outbox event is not retryable")
)

var (
	poolMu sync.RWMutex
	pool   *pgxpool.Pool
)

func SetPool(p *pgxpool.Pool) {
	poolMu.Lock()
	defer poolMu.Unlock()
	pool = p
}

func getPool() *pgxpool.Pool {
	poolMu.RLock()
	defer poolMu.RUnlock()
	return pool
}

func createOutboxEventPostgres(ctx context.Context, evt OutboxEvent) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres outbox store not configured")
	}

	payloadRaw, err := json.Marshal(evt.Payload)
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx, `
		INSERT INTO outbox_events (
			id, trip_id, aggregate_type, aggregate_id, event_type, payload, dedupe_key,
			status, retry_count, available_at, created_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6::jsonb, $7,
			'pending', $8, $9, $10
		)
		ON CONFLICT (dedupe_key) DO NOTHING
	`, evt.ID, evt.TripID, evt.AggregateType, evt.AggregateID, evt.EventType, payloadRaw, evt.DedupeKey, evt.RetryCount, evt.AvailableAt, evt.CreatedAt)
	return err
}

func listOutboxEventsPostgres(ctx context.Context, statusFilter string) ([]OutboxEvent, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres outbox store not configured")
	}

	dbStatus := mapAPIStatusToDB(statusFilter)
	rows, err := p.Query(ctx, `
		SELECT id::text, COALESCE(trip_id::text, ''), aggregate_type, aggregate_id, event_type,
		       payload, dedupe_key, status, retry_count, available_at, processed_at, created_at
		FROM outbox_events
		WHERE status = $1
		  AND available_at <= $2
		ORDER BY created_at ASC
	`, dbStatus, time.Now().UTC())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]OutboxEvent, 0)
	for rows.Next() {
		var (
			item       OutboxEvent
			payloadRaw []byte
			dbStatus   string
			tripID     string
		)
		if err := rows.Scan(
			&item.ID,
			&tripID,
			&item.AggregateType,
			&item.AggregateID,
			&item.EventType,
			&payloadRaw,
			&item.DedupeKey,
			&dbStatus,
			&item.RetryCount,
			&item.AvailableAt,
			&item.ProcessedAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		item.TripID = tripID
		item.Status = mapDBStatusToAPI(dbStatus)
		if len(payloadRaw) > 0 {
			if err := json.Unmarshal(payloadRaw, &item.Payload); err != nil {
				return nil, err
			}
		}
		if item.Payload == nil {
			item.Payload = gin.H{}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func ackOutboxEventPostgres(ctx context.Context, eventID string, success bool) (OutboxEvent, error) {
	p := getPool()
	if p == nil {
		return OutboxEvent{}, errors.New("postgres outbox store not configured")
	}

	for attempt := 1; ; attempt++ {
		item, err := func() (OutboxEvent, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return OutboxEvent{}, err
			}
			defer rollbackTx(ctx, tx)

			var (
				item       OutboxEvent
				payloadRaw []byte
				dbStatus   string
				tripID     string
			)
			err = tx.QueryRow(ctx, `
				SELECT id::text, COALESCE(trip_id::text, ''), aggregate_type, aggregate_id, event_type,
				       payload, dedupe_key, status, retry_count, available_at, processed_at, created_at
				FROM outbox_events
				WHERE id = $1::uuid
				FOR UPDATE
			`, eventID).Scan(
				&item.ID,
				&tripID,
				&item.AggregateType,
				&item.AggregateID,
				&item.EventType,
				&payloadRaw,
				&item.DedupeKey,
				&dbStatus,
				&item.RetryCount,
				&item.AvailableAt,
				&item.ProcessedAt,
				&item.CreatedAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return OutboxEvent{}, ErrOutboxEventNotFound
			}
			if err != nil {
				return OutboxEvent{}, err
			}

			now := time.Now().UTC()
			if success {
				_, err = tx.Exec(ctx, `
					UPDATE outbox_events
					SET status = 'done',
					    processed_at = $2
					WHERE id = $1::uuid
				`, eventID, now)
				if err != nil {
					return OutboxEvent{}, err
				}
				item.Status = "processed"
				item.ProcessedAt = &now
			} else {
				item.RetryCount++
				nextStatus := "pending"
				nextAvailable := item.AvailableAt
				if item.RetryCount > maxRetries {
					nextStatus = "dead"
				} else {
					nextAvailable = now.Add(time.Duration(1<<item.RetryCount) * time.Second)
				}
				_, err = tx.Exec(ctx, `
					UPDATE outbox_events
					SET retry_count = $2,
					    status = $3,
					    available_at = $4
					WHERE id = $1::uuid
				`, eventID, item.RetryCount, nextStatus, nextAvailable)
				if err != nil {
					return OutboxEvent{}, err
				}
				item.Status = mapDBStatusToAPI(nextStatus)
				item.AvailableAt = nextAvailable
			}

			item.TripID = tripID
			if len(payloadRaw) > 0 {
				if err := json.Unmarshal(payloadRaw, &item.Payload); err != nil {
					return OutboxEvent{}, err
				}
			}
			if item.Payload == nil {
				item.Payload = gin.H{}
			}

			if err := tx.Commit(ctx); err != nil {
				return OutboxEvent{}, err
			}
			return item, nil
		}()
		if err == nil {
			return item, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return OutboxEvent{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return OutboxEvent{}, err
	}
}

func retryOutboxEventPostgres(ctx context.Context, eventID string) (OutboxEvent, error) {
	p := getPool()
	if p == nil {
		return OutboxEvent{}, errors.New("postgres outbox store not configured")
	}

	for attempt := 1; ; attempt++ {
		item, err := func() (OutboxEvent, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return OutboxEvent{}, err
			}
			defer rollbackTx(ctx, tx)

			var (
				item       OutboxEvent
				payloadRaw []byte
				dbStatus   string
				tripID     string
			)
			err = tx.QueryRow(ctx, `
				SELECT id::text, COALESCE(trip_id::text, ''), aggregate_type, aggregate_id, event_type,
				       payload, dedupe_key, status, retry_count, available_at, processed_at, created_at
				FROM outbox_events
				WHERE id = $1::uuid
				FOR UPDATE
			`, eventID).Scan(
				&item.ID,
				&tripID,
				&item.AggregateType,
				&item.AggregateID,
				&item.EventType,
				&payloadRaw,
				&item.DedupeKey,
				&dbStatus,
				&item.RetryCount,
				&item.AvailableAt,
				&item.ProcessedAt,
				&item.CreatedAt,
			)
			if errors.Is(err, pgx.ErrNoRows) {
				return OutboxEvent{}, ErrOutboxEventNotFound
			}
			if err != nil {
				return OutboxEvent{}, err
			}
			if dbStatus != "dead" {
				return OutboxEvent{}, ErrOutboxEventNotRetryable
			}

			now := time.Now().UTC()
			_, err = tx.Exec(ctx, `
				UPDATE outbox_events
				SET status = 'pending',
				    retry_count = 0,
				    available_at = $2,
				    processed_at = NULL
				WHERE id = $1::uuid
			`, eventID, now)
			if err != nil {
				return OutboxEvent{}, err
			}

			item.Status = "pending"
			item.RetryCount = 0
			item.AvailableAt = now
			item.ProcessedAt = nil
			item.TripID = tripID
			if len(payloadRaw) > 0 {
				if err := json.Unmarshal(payloadRaw, &item.Payload); err != nil {
					return OutboxEvent{}, err
				}
			}
			if item.Payload == nil {
				item.Payload = gin.H{}
			}

			if err := tx.Commit(ctx); err != nil {
				return OutboxEvent{}, err
			}
			return item, nil
		}()
		if err == nil {
			return item, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return OutboxEvent{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return OutboxEvent{}, err
	}
}

func getOutboxStatsPostgres(ctx context.Context) (OutboxStats, error) {
	p := getPool()
	if p == nil {
		return OutboxStats{}, errors.New("postgres outbox store not configured")
	}

	rows, err := p.Query(ctx, `
		SELECT status, count(*)
		FROM outbox_events
		GROUP BY status
	`)
	if err != nil {
		return OutboxStats{}, err
	}
	defer rows.Close()

	stats := OutboxStats{}
	for rows.Next() {
		var (
			status string
			count  int
		)
		if err := rows.Scan(&status, &count); err != nil {
			return OutboxStats{}, err
		}
		switch status {
		case "pending":
			stats.PendingCount = count
		case "done":
			stats.ProcessedCount = count
		case "dead":
			stats.DLQCount = count
		}
	}
	if err := rows.Err(); err != nil {
		return OutboxStats{}, err
	}
	return stats, nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func mapAPIStatusToDB(status string) string {
	switch status {
	case "processed":
		return "done"
	case "dlq":
		return "dead"
	default:
		return "pending"
	}
}

func mapDBStatusToAPI(status string) string {
	switch status {
	case "done":
		return "processed"
	case "dead":
		return "dlq"
	default:
		return "pending"
	}
}
