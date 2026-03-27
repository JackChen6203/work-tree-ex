package admin

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
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

func listAuditLogsPostgres(ctx context.Context, resourceType, resourceID string) ([]AuditLog, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres admin store not configured")
	}

	query := `
		SELECT id::text,
		       COALESCE(actor_user_id::text, ''),
		       action,
		       resource_type,
		       resource_id,
		       before_state,
		       after_state,
		       COALESCE(request_id, ''),
		       created_at
		FROM audit_logs
		WHERE 1=1
	`
	args := []any{}
	argPos := 1
	if strings.TrimSpace(resourceType) != "" {
		query += " AND resource_type = $" + strconv.Itoa(argPos)
		args = append(args, resourceType)
		argPos++
	}
	if strings.TrimSpace(resourceID) != "" {
		query += " AND resource_id = $" + strconv.Itoa(argPos)
		args = append(args, resourceID)
	}
	query += " ORDER BY created_at DESC LIMIT 500"

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]AuditLog, 0)
	for rows.Next() {
		var (
			item       AuditLog
			actorIDRaw string
			beforeRaw  []byte
			afterRaw   []byte
			requestRaw string
		)
		if err := rows.Scan(
			&item.ID,
			&actorIDRaw,
			&item.Action,
			&item.ResourceType,
			&item.ResourceID,
			&beforeRaw,
			&afterRaw,
			&requestRaw,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}

		if actorIDRaw != "" {
			item.ActorUserID = &actorIDRaw
		}
		if requestRaw != "" {
			item.RequestID = requestRaw
		}
		if len(beforeRaw) > 0 {
			if err := json.Unmarshal(beforeRaw, &item.BeforeState); err != nil {
				return nil, err
			}
		}
		if len(afterRaw) > 0 {
			if err := json.Unmarshal(afterRaw, &item.AfterState); err != nil {
				return nil, err
			}
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func createAuditLogPostgres(ctx context.Context, log AuditLog) (AuditLog, error) {
	p := getPool()
	if p == nil {
		return AuditLog{}, errors.New("postgres admin store not configured")
	}

	var (
		actorID string
		err     error
	)
	if log.ActorUserID != nil {
		actorID = strings.TrimSpace(*log.ActorUserID)
	}

	var beforePayload any
	if log.BeforeState != nil {
		beforePayload, err = json.Marshal(log.BeforeState)
		if err != nil {
			return AuditLog{}, err
		}
	}
	var afterPayload any
	if log.AfterState != nil {
		afterPayload, err = json.Marshal(log.AfterState)
		if err != nil {
			return AuditLog{}, err
		}
	}

	createdAt := log.CreatedAt
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	var out AuditLog
	var actorIDRaw string
	var requestIDRaw string
	var beforeRaw []byte
	var afterRaw []byte

	err = p.QueryRow(ctx, `
		INSERT INTO audit_logs (
			actor_user_id, action, resource_type, resource_id, before_state, after_state, request_id, created_at
		)
		VALUES (
			NULLIF($1, '')::uuid, $2, $3, $4, $5::jsonb, $6::jsonb, NULLIF($7, ''), $8
		)
		RETURNING id::text,
		          COALESCE(actor_user_id::text, ''),
		          action,
		          resource_type,
		          resource_id,
		          before_state,
		          after_state,
		          COALESCE(request_id, ''),
		          created_at
	`, actorID, log.Action, log.ResourceType, log.ResourceID, beforePayload, afterPayload, log.RequestID, createdAt).Scan(
		&out.ID,
		&actorIDRaw,
		&out.Action,
		&out.ResourceType,
		&out.ResourceID,
		&beforeRaw,
		&afterRaw,
		&requestIDRaw,
		&out.CreatedAt,
	)
	if err != nil {
		return AuditLog{}, err
	}

	if actorIDRaw != "" {
		out.ActorUserID = &actorIDRaw
	}
	if requestIDRaw != "" {
		out.RequestID = requestIDRaw
	}
	if len(beforeRaw) > 0 {
		if err := json.Unmarshal(beforeRaw, &out.BeforeState); err != nil {
			return AuditLog{}, err
		}
	}
	if len(afterRaw) > 0 {
		if err := json.Unmarshal(afterRaw, &out.AfterState); err != nil {
			return AuditLog{}, err
		}
	}

	return out, nil
}
