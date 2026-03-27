package ai

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformdb "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
)

var ErrAIDraftNotFound = errors.New("ai draft not found")

const (
	defaultAIUserID           = "00000000-0000-0000-0000-000000000001"
	defaultAIProviderConfigID = "00000000-0000-0000-0000-0000000000a1"
	defaultAIEncryptedKey     = "enc_default_ai_provider_key"
	defaultAIModel            = "gpt-4.1-mini"
)

type aiSummaryPayload struct {
	Summary        string   `json:"summary"`
	Warnings       []string `json:"warnings"`
	TotalEstimated float64  `json:"totalEstimated"`
	Budget         float64  `json:"budget"`
	Currency       string   `json:"currency"`
	Provider       string   `json:"provider"`
}

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

func createPlanPostgres(
	ctx context.Context,
	tripID string,
	in planCreateInput,
	providerName string,
	usage TokenUsage,
	status string,
	warnings []string,
	estimated float64,
) (planDraft, error) {
	p := getPool()
	if p == nil {
		return planDraft{}, errors.New("postgres ai store not configured")
	}

	for attempt := 1; ; attempt++ {
		draft, err := func() (planDraft, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return planDraft{}, err
			}
			defer rollbackAITx(ctx, tx)

			userID, providerConfigID, err := ensureAIPrincipalTx(ctx, tx, strings.TrimSpace(in.ProviderConfigID), providerName)
			if err != nil {
				return planDraft{}, err
			}

			promptContext, err := json.Marshal(ginLikeMap{
				"constraints": in.Constraints,
				"title":       in.Title,
			})
			if err != nil {
				return planDraft{}, err
			}

			queuedAt := time.Now().UTC()
			var requestID string
			if err := tx.QueryRow(ctx, `
				INSERT INTO ai_plan_requests (
					trip_id, requested_by_user_id, provider_config_id, status, prompt_context,
					prompt_tokens, completion_tokens, estimated_cost, queued_at, started_at, finished_at
				)
				VALUES (
					$1::uuid, $2::uuid, $3::uuid, 'succeeded', $4::jsonb,
					$5, $6, $7, $8, $8, $8
				)
				RETURNING id::text
			`, tripID, userID, providerConfigID, promptContext, usage.PromptTokens, usage.CompletionTokens, usage.EstimatedCost, queuedAt).Scan(&requestID); err != nil {
				return planDraft{}, err
			}

			title := strings.TrimSpace(in.Title)
			if title == "" {
				title = "AI Plan Draft"
			}

			draftPayload, err := json.Marshal(ginLikeMap{
				"title":       title,
				"status":      status,
				"constraints": in.Constraints,
				"warnings":    warnings,
				"provider":    providerName,
			})
			if err != nil {
				return planDraft{}, err
			}
			summaryPayload, err := json.Marshal(aiSummaryPayload{
				Summary:        summaryText(estimated, in.Constraints.TotalBudget, in.Constraints.Currency),
				Warnings:       warnings,
				TotalEstimated: estimated,
				Budget:         in.Constraints.TotalBudget,
				Currency:       strings.ToUpper(strings.TrimSpace(in.Constraints.Currency)),
				Provider:       providerName,
			})
			if err != nil {
				return planDraft{}, err
			}

			var (
				draftID   string
				createdAt time.Time
			)
			if err := tx.QueryRow(ctx, `
				INSERT INTO ai_plan_drafts (
					trip_id, request_id, title, status, draft_payload, summary_payload, version, created_at
				)
				VALUES (
					$1::uuid, $2::uuid, $3, $4, $5::jsonb, $6::jsonb, 1, $7
				)
				RETURNING id::text, created_at
			`, tripID, requestID, title, status, draftPayload, summaryPayload, queuedAt).Scan(&draftID, &createdAt); err != nil {
				return planDraft{}, err
			}

			severity := "warning"
			if status == "invalid" {
				severity = "error"
			}
			for _, warning := range warnings {
				if _, err := tx.Exec(ctx, `
					INSERT INTO ai_plan_validation_results (draft_id, severity, rule_code, message, details, created_at)
					VALUES ($1::uuid, $2, $3, $4, '{}'::jsonb, $5)
				`, draftID, severity, "BUDGET_THRESHOLD", warning, queuedAt); err != nil {
					return planDraft{}, err
				}
			}

			if err := tx.Commit(ctx); err != nil {
				return planDraft{}, err
			}

			return planDraft{
				ID:               draftID,
				TripID:           tripID,
				Title:            title,
				Status:           status,
				Summary:          summaryText(estimated, in.Constraints.TotalBudget, in.Constraints.Currency),
				Warnings:         warnings,
				TotalEstimated:   estimated,
				Budget:           in.Constraints.TotalBudget,
				Currency:         strings.ToUpper(strings.TrimSpace(in.Constraints.Currency)),
				PromptTokens:     usage.PromptTokens,
				CompletionTokens: usage.CompletionTokens,
				EstimatedCost:    usage.EstimatedCost,
				Provider:         providerName,
				CreatedAt:        createdAt,
			}, nil
		}()
		if err == nil {
			return draft, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return planDraft{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return planDraft{}, err
	}
}

func recordFailedPlanRequestPostgres(ctx context.Context, tripID string, in planCreateInput, providerName, failureCode, failureMessage string) {
	p := getPool()
	if p == nil {
		return
	}

	userID, providerConfigID, err := ensureAIPrincipal(ctx, p, strings.TrimSpace(in.ProviderConfigID), providerName)
	if err != nil {
		return
	}

	promptContext, err := json.Marshal(ginLikeMap{
		"constraints": in.Constraints,
		"title":       in.Title,
	})
	if err != nil {
		return
	}

	now := time.Now().UTC()
	_, _ = p.Exec(ctx, `
		INSERT INTO ai_plan_requests (
			trip_id, requested_by_user_id, provider_config_id, status, prompt_context,
			queued_at, started_at, finished_at, failure_code, failure_message
		)
		VALUES (
			$1::uuid, $2::uuid, $3::uuid, 'failed', $4::jsonb,
			$5, $5, $5, $6, $7
		)
	`, tripID, userID, providerConfigID, promptContext, now, failureCode, failureMessage)
}

func listPlansPostgres(ctx context.Context, tripID string) ([]planDraft, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres ai store not configured")
	}

	rows, err := p.Query(ctx, `
		SELECT d.id::text,
		       d.trip_id::text,
		       d.title,
		       d.status,
		       d.summary_payload,
		       d.created_at,
		       COALESCE(r.prompt_tokens, 0),
		       COALESCE(r.completion_tokens, 0),
		       COALESCE(r.estimated_cost::float8, 0),
		       COALESCE(cfg.provider, 'openai')
		FROM ai_plan_drafts d
		JOIN ai_plan_requests r ON r.id = d.request_id
		LEFT JOIN llm_provider_configs cfg ON cfg.id = r.provider_config_id
		WHERE d.trip_id = $1::uuid
		ORDER BY d.created_at DESC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]planDraft, 0)
	for rows.Next() {
		item, err := scanPlanDraft(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func getPlanPostgres(ctx context.Context, tripID, planID string) (planDraft, error) {
	p := getPool()
	if p == nil {
		return planDraft{}, errors.New("postgres ai store not configured")
	}

	item, err := scanPlanDraftRow(p.QueryRow(ctx, `
		SELECT d.id::text,
		       d.trip_id::text,
		       d.title,
		       d.status,
		       d.summary_payload,
		       d.created_at,
		       COALESCE(r.prompt_tokens, 0),
		       COALESCE(r.completion_tokens, 0),
		       COALESCE(r.estimated_cost::float8, 0),
		       COALESCE(cfg.provider, 'openai')
		FROM ai_plan_drafts d
		JOIN ai_plan_requests r ON r.id = d.request_id
		LEFT JOIN llm_provider_configs cfg ON cfg.id = r.provider_config_id
		WHERE d.trip_id = $1::uuid
		  AND d.id = $2::uuid
	`, tripID, planID))
	if errors.Is(err, pgx.ErrNoRows) {
		return planDraft{}, ErrAIDraftNotFound
	}
	if err != nil {
		return planDraft{}, err
	}
	return item, nil
}

func ensureAIPrincipal(ctx context.Context, p *pgxpool.Pool, requestedProviderConfigID, providerName string) (string, string, error) {
	for attempt := 1; ; attempt++ {
		userID, providerConfigID, err := func() (string, string, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return "", "", err
			}
			defer rollbackAITx(ctx, tx)

			userID, providerConfigID, err := ensureAIPrincipalTx(ctx, tx, requestedProviderConfigID, providerName)
			if err != nil {
				return "", "", err
			}
			if err := tx.Commit(ctx); err != nil {
				return "", "", err
			}
			return userID, providerConfigID, nil
		}()
		if err == nil {
			return userID, providerConfigID, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return "", "", platformdb.DeadlockRetryExhaustedError(err)
		}
		return "", "", err
	}
}

func ensureAIPrincipalTx(ctx context.Context, tx pgx.Tx, requestedProviderConfigID, providerName string) (string, string, error) {
	_, err := tx.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (id) DO NOTHING
	`, defaultAIUserID, "system@time-tree.local", "System")
	if err != nil {
		return "", "", err
	}

	providerConfigID := requestedProviderConfigID
	if _, err := uuid.Parse(providerConfigID); err != nil {
		providerConfigID = defaultAIProviderConfigID
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO llm_provider_configs (
			id, user_id, provider, label, encrypted_key, model, is_active, created_at
		)
		VALUES (
			$1::uuid, $2::uuid, $3, 'system-default', $4, $5, true, now()
		)
		ON CONFLICT (id) DO NOTHING
	`, providerConfigID, defaultAIUserID, strings.TrimSpace(providerName), defaultAIEncryptedKey, defaultAIModel)
	if err != nil {
		return "", "", err
	}

	return defaultAIUserID, providerConfigID, nil
}

type planDraftScanner interface {
	Scan(dest ...any) error
}

func scanPlanDraft(scanner planDraftScanner) (planDraft, error) {
	return scanPlanDraftRow(scanner)
}

func scanPlanDraftRow(scanner planDraftScanner) (planDraft, error) {
	var (
		item       planDraft
		summaryRaw []byte
		summary    aiSummaryPayload
	)
	if err := scanner.Scan(
		&item.ID,
		&item.TripID,
		&item.Title,
		&item.Status,
		&summaryRaw,
		&item.CreatedAt,
		&item.PromptTokens,
		&item.CompletionTokens,
		&item.EstimatedCost,
		&item.Provider,
	); err != nil {
		return planDraft{}, err
	}
	if len(summaryRaw) > 0 {
		if err := json.Unmarshal(summaryRaw, &summary); err != nil {
			return planDraft{}, err
		}
	}
	item.Summary = summary.Summary
	item.Warnings = summary.Warnings
	item.TotalEstimated = summary.TotalEstimated
	item.Budget = summary.Budget
	item.Currency = summary.Currency
	if item.Provider == "" {
		item.Provider = summary.Provider
	}
	return item, nil
}

func summaryText(estimated, budget float64, currency string) string {
	return "Estimated " + trimFloat(estimated) + " " + strings.ToUpper(strings.TrimSpace(currency)) +
		" against budget " + trimFloat(budget) + " " + strings.ToUpper(strings.TrimSpace(currency))
}

func trimFloat(v float64) string {
	return strings.TrimRight(strings.TrimRight(fmtFloat(v), "0"), ".")
}

func fmtFloat(v float64) string {
	return strconv.FormatFloat(v, 'f', 2, 64)
}

type ginLikeMap map[string]any

func rollbackAITx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}

func adoptDraftToItineraryPostgres(ctx context.Context, tripID, planID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres ai store not configured")
	}

	for attempt := 1; ; attempt++ {
		err := func() error {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return err
			}
			defer rollbackAITx(ctx, tx)

			var draftTitle string
			err = tx.QueryRow(ctx, `
				SELECT title
				FROM ai_plan_drafts
				WHERE id = $1::uuid
				  AND trip_id = $2::uuid
				FOR UPDATE
			`, planID, tripID).Scan(&draftTitle)
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrAIDraftNotFound
			}
			if err != nil {
				return err
			}

			if err := ensureItineraryDaysTx(ctx, tx, tripID); err != nil {
				return err
			}

			var dayID string
			err = tx.QueryRow(ctx, `
				SELECT id::text
				FROM itinerary_days
				WHERE trip_id = $1::uuid
				ORDER BY sort_order ASC, day_index ASC
				LIMIT 1
			`, tripID).Scan(&dayID)
			if err != nil {
				return err
			}

			var nextSort int
			if err := tx.QueryRow(ctx, `
				SELECT COALESCE(MAX(sort_order), 0) + 1
				FROM itinerary_items
				WHERE trip_id = $1::uuid
				  AND day_id = $2::uuid
				  AND deleted_at IS NULL
			`, tripID, dayID).Scan(&nextSort); err != nil {
				return err
			}

			if _, err := tx.Exec(ctx, `
				INSERT INTO itinerary_items (
					trip_id, day_id, title, item_type, all_day, sort_order, source_type, source_draft_id, version, created_at, updated_at
				)
				VALUES (
					$1::uuid, $2::uuid, $3, 'custom', true, $4, 'ai_draft', $5::uuid, 1, now(), now()
				)
			`, tripID, dayID, "[AI] "+strings.TrimSpace(draftTitle), nextSort, planID); err != nil {
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

func ensureItineraryDaysTx(ctx context.Context, tx pgx.Tx, tripID string) error {
	var dayCount int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(1)
		FROM itinerary_days
		WHERE trip_id = $1::uuid
	`, tripID).Scan(&dayCount); err != nil {
		return err
	}
	if dayCount > 0 {
		return nil
	}

	var (
		startDate time.Time
		endDate   time.Time
	)
	if err := tx.QueryRow(ctx, `
		SELECT start_date, end_date
		FROM trips
		WHERE id = $1::uuid
	`, tripID).Scan(&startDate, &endDate); err != nil {
		return err
	}

	dayIndex := 1
	for d := startDate; !d.After(endDate); d = d.AddDate(0, 0, 1) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO itinerary_days (trip_id, trip_date, day_index, sort_order, version, created_at, updated_at)
			VALUES ($1::uuid, $2::date, $3, $4, 1, now(), now())
			ON CONFLICT (trip_id, trip_date) DO NOTHING
		`, tripID, d.Format("2006-01-02"), dayIndex, dayIndex); err != nil {
			return err
		}
		dayIndex++
	}
	return nil
}

func writeAIAuditLogPostgres(ctx context.Context, action, resourceType, resourceID string, beforeState, afterState any) error {
	p := getPool()
	if p == nil {
		return nil
	}

	var beforePayload any
	if beforeState != nil {
		raw, err := json.Marshal(beforeState)
		if err != nil {
			return err
		}
		beforePayload = raw
	}
	var afterPayload any
	if afterState != nil {
		raw, err := json.Marshal(afterState)
		if err != nil {
			return err
		}
		afterPayload = raw
	}

	_, err := p.Exec(ctx, `
		INSERT INTO audit_logs (
			actor_user_id, action, resource_type, resource_id, before_state, after_state, request_id, created_at
		)
		VALUES (
			NULL, $1, $2, $3, $4::jsonb, $5::jsonb, NULL, now()
		)
	`, action, resourceType, resourceID, beforePayload, afterPayload)
	return err
}
