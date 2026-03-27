package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformdb "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
)

var (
	ErrProviderNotFound          = errors.New("provider not found")
	ErrPreferenceVersionConflict = errors.New("preference version conflict")
)

const defaultUserID = "00000000-0000-0000-0000-000000000001"

const (
	defaultUserEmail       = "ariel@example.com"
	defaultUserDisplayName = "Ariel Chen"
	defaultUserLocale      = "zh-TW"
	defaultUserTimezone    = "Asia/Taipei"
	defaultUserCurrency    = "TWD"
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

func getProfilePostgres(ctx context.Context) (profile, error) {
	p := getPool()
	if p == nil {
		return profile{}, errors.New("postgres users store not configured")
	}

	if err := ensureDefaultUser(ctx, p); err != nil {
		return profile{}, err
	}

	var item profile
	err := p.QueryRow(ctx, `
		SELECT id::text, email, display_name, locale, timezone, default_currency
		FROM users
		WHERE id = $1::uuid
	`, defaultUserID).Scan(
		&item.ID,
		&item.Email,
		&item.DisplayName,
		&item.Locale,
		&item.Timezone,
		&item.Currency,
	)
	return item, err
}

func patchProfilePostgres(ctx context.Context, in profilePatchInput) (profile, error) {
	p := getPool()
	if p == nil {
		return profile{}, errors.New("postgres users store not configured")
	}

	for attempt := 1; ; attempt++ {
		updated, err := func() (profile, error) {
			tx, err := p.Begin(ctx)
			if err != nil {
				return profile{}, err
			}
			defer rollbackUsersTx(ctx, tx)

			if err := ensureDefaultUser(ctx, p); err != nil {
				return profile{}, err
			}

			var current profile
			err = tx.QueryRow(ctx, `
				SELECT id::text, email, display_name, locale, timezone, default_currency
				FROM users
				WHERE id = $1::uuid
				FOR UPDATE
			`, defaultUserID).Scan(
				&current.ID,
				&current.Email,
				&current.DisplayName,
				&current.Locale,
				&current.Timezone,
				&current.Currency,
			)
			if err != nil {
				return profile{}, err
			}

			if in.DisplayName != nil {
				current.DisplayName = strings.TrimSpace(*in.DisplayName)
			}
			if in.Locale != nil {
				current.Locale = strings.TrimSpace(*in.Locale)
			}
			if in.Timezone != nil {
				current.Timezone = strings.TrimSpace(*in.Timezone)
			}
			if in.Currency != nil {
				current.Currency = strings.ToUpper(strings.TrimSpace(*in.Currency))
			}

			var updated profile
			err = tx.QueryRow(ctx, `
				UPDATE users
				SET display_name = $2,
				    locale = $3,
				    timezone = $4,
				    default_currency = $5,
				    updated_at = now()
				WHERE id = $1::uuid
				RETURNING id::text, email, display_name, locale, timezone, default_currency
			`, defaultUserID, current.DisplayName, current.Locale, current.Timezone, current.Currency).Scan(
				&updated.ID,
				&updated.Email,
				&updated.DisplayName,
				&updated.Locale,
				&updated.Timezone,
				&updated.Currency,
			)
			if err != nil {
				return profile{}, err
			}

			if err := tx.Commit(ctx); err != nil {
				return profile{}, err
			}
			return updated, nil
		}()
		if err == nil {
			return updated, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return profile{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return profile{}, err
	}
}

func getPreferencePostgres(ctx context.Context) (preference, error) {
	p := getPool()
	if p == nil {
		return preference{}, errors.New("postgres users store not configured")
	}

	if err := ensureDefaultUser(ctx, p); err != nil {
		return preference{}, err
	}
	if err := ensureDefaultPreference(ctx, p); err != nil {
		return preference{}, err
	}

	return loadPreferencePostgres(ctx, p)
}

func putPreferencePostgres(ctx context.Context, in preference, expectedVersion *int) (preference, error) {
	p := getPool()
	if p == nil {
		return preference{}, errors.New("postgres users store not configured")
	}

	for attempt := 1; ; attempt++ {
		out, err := func() (preference, error) {
			tx, err := p.Begin(ctx)
			if err != nil {
				return preference{}, err
			}
			defer rollbackUsersTx(ctx, tx)

			if err := ensureDefaultUser(ctx, p); err != nil {
				return preference{}, err
			}
			if err := ensureDefaultPreferenceTx(ctx, tx); err != nil {
				return preference{}, err
			}

			var currentVersion int
			err = tx.QueryRow(ctx, `
				SELECT version
				FROM user_preferences
				WHERE user_id = $1::uuid
				FOR UPDATE
			`, defaultUserID).Scan(&currentVersion)
			if err != nil {
				return preference{}, err
			}

			if expectedVersion != nil && *expectedVersion != currentVersion {
				return preference{}, fmt.Errorf("%w: currentVersion=%d", ErrPreferenceVersionConflict, currentVersion)
			}

			nextVersion := currentVersion + 1
			payload, err := explicitPreferenceJSON(in)
			if err != nil {
				return preference{}, err
			}

			_, err = tx.Exec(ctx, `
				UPDATE user_preferences
				SET explicit_preferences = $2::jsonb,
				    version = $3,
				    updated_at = now()
				WHERE user_id = $1::uuid
			`, defaultUserID, payload, nextVersion)
			if err != nil {
				return preference{}, err
			}

			out := in
			out.Version = nextVersion

			if err := tx.Commit(ctx); err != nil {
				return preference{}, err
			}
			return out, nil
		}()
		if err == nil {
			return out, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return preference{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return preference{}, err
	}
}

func clearUserDataPostgres(ctx context.Context) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres users store not configured")
	}

	for attempt := 1; ; attempt++ {
		err := func() error {
			tx, err := p.Begin(ctx)
			if err != nil {
				return err
			}
			defer rollbackUsersTx(ctx, tx)

			if err := ensureDefaultUser(ctx, p); err != nil {
				return err
			}

			_, err = tx.Exec(ctx, `
				DELETE FROM llm_provider_configs
				WHERE user_id = $1::uuid
			`, defaultUserID)
			if err != nil {
				return err
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO user_preferences (user_id, explicit_preferences, inferred_preferences, version, updated_at)
				VALUES ($1::uuid, '{}'::jsonb, '{}'::jsonb, 0, now())
				ON CONFLICT (user_id)
				DO UPDATE SET
					explicit_preferences = '{}'::jsonb,
					inferred_preferences = '{}'::jsonb,
					version = 0,
					updated_at = now()
			`, defaultUserID)
			if err != nil {
				return err
			}

			deletedEmail := "deleted+" + strings.ReplaceAll(defaultUserID, "-", "") + "@time-tree.local"
			_, err = tx.Exec(ctx, `
				UPDATE users
				SET email = $2,
				    display_name = '[deleted]',
				    locale = '',
				    timezone = '',
				    default_currency = '',
				    updated_at = now()
				WHERE id = $1::uuid
			`, defaultUserID, deletedEmail)
			if err != nil {
				return err
			}

			_, err = tx.Exec(ctx, `
				INSERT INTO audit_logs (
					actor_user_id, action, resource_type, resource_id, before_state, after_state, request_id, created_at
				)
				VALUES (
					$1::uuid, 'delete_user', 'users', $2, '{}'::jsonb, '{}'::jsonb, 'system', now()
				)
			`, defaultUserID, defaultUserID)
			if err != nil {
				return err
			}

			return tx.Commit(ctx)
		}()
		if err == nil {
			return nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return platformdb.DeadlockRetryExhaustedError(err)
		}
		return err
	}
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

func ensureDefaultUser(ctx context.Context, p *pgxpool.Pool) error {
	_, err := p.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, $4, $5, $6, now(), now())
		ON CONFLICT (id) DO NOTHING
	`, defaultUserID, defaultUserEmail, defaultUserDisplayName, defaultUserLocale, defaultUserTimezone, defaultUserCurrency)
	return err
}

func ensureDefaultPreference(ctx context.Context, p *pgxpool.Pool) error {
	payload, err := explicitPreferenceJSON(preference{
		TripPace:            "balanced",
		WakePattern:         "normal",
		TransportPreference: "transit",
		FoodPreference:      []string{"coffee", "local"},
		AvoidTags:           []string{"too-many-transfers"},
	})
	if err != nil {
		return err
	}

	_, err = p.Exec(ctx, `
		INSERT INTO user_preferences (user_id, explicit_preferences, inferred_preferences, version, updated_at)
		VALUES ($1::uuid, $2::jsonb, '{}'::jsonb, 1, now())
		ON CONFLICT (user_id) DO NOTHING
	`, defaultUserID, payload)
	return err
}

func ensureDefaultPreferenceTx(ctx context.Context, tx pgx.Tx) error {
	payload, err := explicitPreferenceJSON(preference{
		TripPace:            "balanced",
		WakePattern:         "normal",
		TransportPreference: "transit",
		FoodPreference:      []string{"coffee", "local"},
		AvoidTags:           []string{"too-many-transfers"},
	})
	if err != nil {
		return err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO user_preferences (user_id, explicit_preferences, inferred_preferences, version, updated_at)
		VALUES ($1::uuid, $2::jsonb, '{}'::jsonb, 1, now())
		ON CONFLICT (user_id) DO NOTHING
	`, defaultUserID, payload)
	return err
}

func loadPreferencePostgres(ctx context.Context, p *pgxpool.Pool) (preference, error) {
	var (
		explicitRaw []byte
		item        preference
	)
	err := p.QueryRow(ctx, `
		SELECT explicit_preferences, version
		FROM user_preferences
		WHERE user_id = $1::uuid
	`, defaultUserID).Scan(&explicitRaw, &item.Version)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return preference{Version: 0}, nil
		}
		return preference{}, err
	}

	var payload struct {
		TripPace            string   `json:"tripPace"`
		WakePattern         string   `json:"wakePattern"`
		TransportPreference string   `json:"transportPreference"`
		FoodPreference      []string `json:"foodPreference"`
		AvoidTags           []string `json:"avoidTags"`
	}
	if len(explicitRaw) > 0 {
		if err := json.Unmarshal(explicitRaw, &payload); err != nil {
			return preference{}, err
		}
	}

	item.TripPace = payload.TripPace
	item.WakePattern = payload.WakePattern
	item.TransportPreference = payload.TransportPreference
	item.FoodPreference = payload.FoodPreference
	item.AvoidTags = payload.AvoidTags
	return item, nil
}

func explicitPreferenceJSON(in preference) ([]byte, error) {
	payload := struct {
		TripPace            string   `json:"tripPace"`
		WakePattern         string   `json:"wakePattern"`
		TransportPreference string   `json:"transportPreference"`
		FoodPreference      []string `json:"foodPreference"`
		AvoidTags           []string `json:"avoidTags"`
	}{
		TripPace:            in.TripPace,
		WakePattern:         in.WakePattern,
		TransportPreference: in.TransportPreference,
		FoodPreference:      in.FoodPreference,
		AvoidTags:           in.AvoidTags,
	}
	return json.Marshal(payload)
}

func rollbackUsersTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func maskEnvelope(envelope string) string {
	if len(envelope) <= 4 {
		return "****"
	}
	return "****" + envelope[len(envelope)-4:]
}
