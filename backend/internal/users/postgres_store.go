package users

import (
	"context"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrProviderNotFound = errors.New("provider not found")

const defaultUserID = "00000000-0000-0000-0000-000000000001"

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

func listProvidersPostgres(ctx context.Context, providerFilter string) ([]llmProvider, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres users store not configured")
	}

	query := `
		SELECT id::text, provider, label, model, encrypted_key, created_at
		FROM llm_provider_configs
		WHERE user_id = $1::uuid
		  AND is_active = true
	`
	args := []any{defaultUserID}
	if providerFilter != "" {
		query += " AND provider = $2"
		args = append(args, providerFilter)
	}
	query += " ORDER BY created_at DESC"

	rows, err := p.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]llmProvider, 0)
	for rows.Next() {
		var (
			item         llmProvider
			encryptedKey string
		)
		if err := rows.Scan(&item.ID, &item.Provider, &item.Label, &item.Model, &encryptedKey, &item.CreatedAt); err != nil {
			return nil, err
		}
		item.MaskedKey = maskEnvelope(encryptedKey)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func createProviderPostgres(ctx context.Context, in llmProviderInput) (llmProvider, error) {
	p := getPool()
	if p == nil {
		return llmProvider{}, errors.New("postgres users store not configured")
	}

	if err := ensureDefaultUser(ctx, p); err != nil {
		return llmProvider{}, err
	}

	envelope := strings.TrimSpace(in.EncryptedAPIKeyEnvelope)
	item := llmProvider{}
	err := p.QueryRow(ctx, `
		INSERT INTO llm_provider_configs (
			user_id, provider, label, encrypted_key, model, is_active, created_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5, true, $6
		)
		RETURNING id::text, provider, label, model, created_at
	`, defaultUserID, strings.TrimSpace(in.Provider), strings.TrimSpace(in.Label), envelope, strings.TrimSpace(in.Model), time.Now().UTC()).Scan(
		&item.ID,
		&item.Provider,
		&item.Label,
		&item.Model,
		&item.CreatedAt,
	)
	if err != nil {
		return llmProvider{}, err
	}
	item.MaskedKey = maskEnvelope(envelope)
	return item, nil
}

func deleteProviderPostgres(ctx context.Context, providerID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres users store not configured")
	}

	res, err := p.Exec(ctx, `
		DELETE FROM llm_provider_configs
		WHERE id = $1::uuid
		  AND user_id = $2::uuid
	`, providerID, defaultUserID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrProviderNotFound
	}
	return nil
}

func clearProvidersPostgres(ctx context.Context) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres users store not configured")
	}
	_, err := p.Exec(ctx, `
		DELETE FROM llm_provider_configs
		WHERE user_id = $1::uuid
	`, defaultUserID)
	return err
}

func ensureDefaultUser(ctx context.Context, p *pgxpool.Pool) error {
	_, err := p.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (id) DO NOTHING
	`, defaultUserID, "system@time-tree.local", "System")
	return err
}

func maskEnvelope(envelope string) string {
	if len(envelope) <= 4 {
		return "****"
	}
	return "****" + envelope[len(envelope)-4:]
}
