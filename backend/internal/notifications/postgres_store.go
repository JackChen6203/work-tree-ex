package notifications

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrNotificationNotFound = errors.New("notification not found")
	ErrCursorNotFound       = errors.New("cursor not found")
)

const defaultNotificationUserID = "00000000-0000-0000-0000-000000000001"

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

func listNotificationsPostgres(ctx context.Context, unreadOnly bool, cursor string, limit int) ([]notification, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres notifications store not configured")
	}

	query := `
		SELECT id::text, type, title, body, COALESCE(link, ''), read_at, created_at
		FROM notifications
	`
	args := []any{}
	if unreadOnly {
		query += "WHERE read_at IS NULL "
	}
	query += "ORDER BY created_at DESC, id DESC"

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	all := make([]notification, 0)
	for rows.Next() {
		var item notification
		if err := rows.Scan(
			&item.ID,
			&item.Type,
			&item.Title,
			&item.Body,
			&item.Link,
			&item.ReadAt,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		all = append(all, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	start := 0
	if cursor != "" {
		start = -1
		for i := range all {
			if all[i].ID == cursor {
				start = i + 1
				break
			}
		}
		if start == -1 {
			return nil, ErrCursorNotFound
		}
	}

	items := make([]notification, 0, limit)
	for i := start; i < len(all); i++ {
		items = append(items, all[i])
		if len(items) == limit {
			break
		}
	}
	return items, nil
}

func markReadPostgres(ctx context.Context, notificationID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres notifications store not configured")
	}

	res, err := p.Exec(ctx, `
		UPDATE notifications
		SET read_at = $2
		WHERE id = $1::uuid
	`, notificationID, time.Now().UTC())
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func markUnreadPostgres(ctx context.Context, notificationID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres notifications store not configured")
	}

	res, err := p.Exec(ctx, `
		UPDATE notifications
		SET read_at = NULL
		WHERE id = $1::uuid
	`, notificationID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func markAllReadPostgres(ctx context.Context) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres notifications store not configured")
	}
	_, err := p.Exec(ctx, `
		UPDATE notifications
		SET read_at = $1
		WHERE read_at IS NULL
	`, time.Now().UTC())
	return err
}

func cleanupReadPostgres(ctx context.Context) (int, error) {
	p := getPool()
	if p == nil {
		return 0, errors.New("postgres notifications store not configured")
	}
	res, err := p.Exec(ctx, `
		DELETE FROM notifications
		WHERE read_at IS NOT NULL
	`)
	if err != nil {
		return 0, err
	}
	return int(res.RowsAffected()), nil
}

func deleteNotificationPostgres(ctx context.Context, notificationID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres notifications store not configured")
	}
	res, err := p.Exec(ctx, `
		DELETE FROM notifications
		WHERE id = $1::uuid
	`, notificationID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrNotificationNotFound
	}
	return nil
}

func createNotificationPostgres(ctx context.Context, in triggerInput, now time.Time) (string, error) {
	p := getPool()
	if p == nil {
		return "", errors.New("postgres notifications store not configured")
	}

	userID := stringsTrimOrDefault(in.UserID, defaultNotificationUserID)
	if _, err := uuid.Parse(userID); err != nil {
		userID = defaultNotificationUserID
	}

	if err := ensureNotificationUser(ctx, p, userID); err != nil {
		return "", err
	}

	var id string
	err := p.QueryRow(ctx, `
		INSERT INTO notifications (user_id, type, title, body, payload, link, created_at)
		VALUES ($1::uuid, $2, $3, $4, '{}'::jsonb, NULLIF($5, ''), $6)
		RETURNING id::text
	`, userID, in.EventType, in.Title, in.Body, in.Link, now).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func ensureNotificationUser(ctx context.Context, p *pgxpool.Pool, userID string) error {
	email := "user-" + userID + "@time-tree.local"
	if userID == defaultNotificationUserID {
		email = "system@time-tree.local"
	}
	_, err := p.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (id) DO NOTHING
	`, userID, email, "System")
	return err
}

func stringsTrimOrDefault(v, fallback string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return fallback
	}
	return v
}
