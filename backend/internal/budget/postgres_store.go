package budget

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	ErrExpenseNotFound = errors.New("expense not found")
)

const defaultBudgetUserID = "00000000-0000-0000-0000-000000000001"

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

func getBudgetProfilePostgres(ctx context.Context, tripID string) (budgetProfile, bool, error) {
	p := getPool()
	if p == nil {
		return budgetProfile{}, false, errors.New("postgres budget store not configured")
	}

	var (
		profile       budgetProfile
		categoriesRaw []byte
	)
	err := p.QueryRow(ctx, `
		SELECT trip_id::text,
		       total_budget::float8,
		       per_person_budget::float8,
		       per_day_budget::float8,
		       currency,
		       category_plan,
		       version,
		       created_at,
		       updated_at
		FROM budget_profiles
		WHERE trip_id = $1::uuid
	`, tripID).Scan(
		&profile.TripID,
		&profile.TotalBudget,
		&profile.PerPersonBudget,
		&profile.PerDayBudget,
		&profile.Currency,
		&categoriesRaw,
		&profile.Version,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return budgetProfile{}, false, nil
	}
	if err != nil {
		return budgetProfile{}, false, err
	}

	if len(categoriesRaw) > 0 {
		if err := json.Unmarshal(categoriesRaw, &profile.Categories); err != nil {
			return budgetProfile{}, false, err
		}
	}
	return profile, true, nil
}

func upsertBudgetPostgres(ctx context.Context, tripID string, in budgetProfileInput) (budgetProfile, error) {
	p := getPool()
	if p == nil {
		return budgetProfile{}, errors.New("postgres budget store not configured")
	}

	categoriesRaw, err := json.Marshal(in.Categories)
	if err != nil {
		return budgetProfile{}, err
	}

	now := time.Now().UTC()
	var (
		profile       budgetProfile
		categoriesOut []byte
	)
	err = p.QueryRow(ctx, `
		INSERT INTO budget_profiles (
			trip_id, total_budget, per_person_budget, per_day_budget, currency, category_plan, version, created_at, updated_at
		) VALUES (
			$1::uuid, $2, $3, $4, $5, $6::jsonb, 1, $7, $7
		)
		ON CONFLICT (trip_id)
		DO UPDATE SET
			total_budget = EXCLUDED.total_budget,
			per_person_budget = EXCLUDED.per_person_budget,
			per_day_budget = EXCLUDED.per_day_budget,
			currency = EXCLUDED.currency,
			category_plan = EXCLUDED.category_plan,
			version = budget_profiles.version + 1,
			updated_at = EXCLUDED.updated_at
		RETURNING trip_id::text,
		          total_budget::float8,
		          per_person_budget::float8,
		          per_day_budget::float8,
		          currency,
		          category_plan,
		          version,
		          created_at,
		          updated_at
	`, tripID, in.TotalBudget, in.PerPersonBudget, in.PerDayBudget, strings.ToUpper(in.Currency), categoriesRaw, now).Scan(
		&profile.TripID,
		&profile.TotalBudget,
		&profile.PerPersonBudget,
		&profile.PerDayBudget,
		&profile.Currency,
		&categoriesOut,
		&profile.Version,
		&profile.CreatedAt,
		&profile.UpdatedAt,
	)
	if err != nil {
		return budgetProfile{}, err
	}

	if len(categoriesOut) > 0 {
		if err := json.Unmarshal(categoriesOut, &profile.Categories); err != nil {
			return budgetProfile{}, err
		}
	}
	return profile, nil
}

func getActualSpendPostgres(ctx context.Context, tripID string) (float64, error) {
	p := getPool()
	if p == nil {
		return 0, errors.New("postgres budget store not configured")
	}

	var actual float64
	if err := p.QueryRow(ctx, `
		SELECT COALESCE(SUM(amount)::float8, 0)
		FROM expenses
		WHERE trip_id = $1::uuid
		  AND deleted_at IS NULL
	`, tripID).Scan(&actual); err != nil {
		return 0, err
	}
	return actual, nil
}

func listExpensesPostgres(ctx context.Context, tripID string) ([]expense, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres budget store not configured")
	}

	rows, err := p.Query(ctx, `
		SELECT id::text,
		       trip_id::text,
		       category,
		       amount::float8,
		       currency,
		       expense_at,
		       COALESCE(note, ''),
		       COALESCE(linked_item_id::text, ''),
		       created_at
		FROM expenses
		WHERE trip_id = $1::uuid
		  AND deleted_at IS NULL
		ORDER BY created_at ASC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]expense, 0)
	for rows.Next() {
		var (
			item         expense
			expenseAtVal *time.Time
			linkedItem   string
		)
		if err := rows.Scan(
			&item.ID,
			&item.TripID,
			&item.Category,
			&item.Amount,
			&item.Currency,
			&expenseAtVal,
			&item.Note,
			&linkedItem,
			&item.CreatedAt,
		); err != nil {
			return nil, err
		}
		if expenseAtVal != nil {
			val := expenseAtVal.UTC().Format(time.RFC3339)
			item.ExpenseAt = &val
		}
		if linkedItem != "" {
			item.LinkedItemID = &linkedItem
		}
		if v, ok := expenseVersionByID[item.ID]; ok {
			item.Version = v
		} else {
			item.Version = 1
			expenseVersionByID[item.ID] = 1
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func createExpensePostgres(ctx context.Context, tripID string, in expenseInput) (expense, error) {
	p := getPool()
	if p == nil {
		return expense{}, errors.New("postgres budget store not configured")
	}

	if err := ensureBudgetUser(ctx, p, defaultBudgetUserID); err != nil {
		return expense{}, err
	}

	var (
		expenseAt *time.Time
		err       error
	)
	if in.ExpenseAt != nil {
		expenseAt, err = parseExpenseTime(*in.ExpenseAt)
		if err != nil {
			return expense{}, err
		}
	}

	note := strings.TrimSpace(in.Note)
	linkedItemID := ""
	if in.LinkedItemID != nil {
		linkedItemID = strings.TrimSpace(*in.LinkedItemID)
	}

	item := expense{
		TripID:       tripID,
		Category:     strings.TrimSpace(in.Category),
		Amount:       in.Amount,
		Currency:     strings.ToUpper(strings.TrimSpace(in.Currency)),
		Note:         note,
		LinkedItemID: nil,
	}
	if linkedItemID != "" {
		item.LinkedItemID = &linkedItemID
	}
	if in.ExpenseAt != nil {
		val := strings.TrimSpace(*in.ExpenseAt)
		if val != "" {
			item.ExpenseAt = &val
		}
	}

	var linkedItem any
	if linkedItemID != "" {
		linkedItem = linkedItemID
	}

	err = p.QueryRow(ctx, `
		INSERT INTO expenses (
			trip_id, created_by_user_id, category, amount, currency, expense_at, note, linked_item_id, created_at, updated_at
		) VALUES (
			$1::uuid, $2::uuid, $3, $4, $5, $6, NULLIF($7, ''), $8::uuid, now(), now()
		)
		RETURNING id::text, created_at
	`, tripID, defaultBudgetUserID, item.Category, item.Amount, item.Currency, expenseAt, note, linkedItem).Scan(
		&item.ID,
		&item.CreatedAt,
	)
	if err != nil {
		return expense{}, err
	}

	item.Version = 1
	expenseVersionByID[item.ID] = 1
	return item, nil
}

func getExpensePostgres(ctx context.Context, tripID, expenseID string) (expense, error) {
	p := getPool()
	if p == nil {
		return expense{}, errors.New("postgres budget store not configured")
	}

	var (
		item         expense
		expenseAtVal *time.Time
		linkedItem   string
	)
	err := p.QueryRow(ctx, `
		SELECT id::text,
		       trip_id::text,
		       category,
		       amount::float8,
		       currency,
		       expense_at,
		       COALESCE(note, ''),
		       COALESCE(linked_item_id::text, ''),
		       created_at
		FROM expenses
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, tripID, expenseID).Scan(
		&item.ID,
		&item.TripID,
		&item.Category,
		&item.Amount,
		&item.Currency,
		&expenseAtVal,
		&item.Note,
		&linkedItem,
		&item.CreatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return expense{}, ErrExpenseNotFound
	}
	if err != nil {
		return expense{}, err
	}

	if expenseAtVal != nil {
		val := expenseAtVal.UTC().Format(time.RFC3339)
		item.ExpenseAt = &val
	}
	if linkedItem != "" {
		item.LinkedItemID = &linkedItem
	}
	if v, ok := expenseVersionByID[item.ID]; ok {
		item.Version = v
	} else {
		item.Version = 1
		expenseVersionByID[item.ID] = 1
	}
	return item, nil
}

func patchExpensePostgres(ctx context.Context, tripID, expenseID string, in expensePatchInput) (expense, error) {
	p := getPool()
	if p == nil {
		return expense{}, errors.New("postgres budget store not configured")
	}

	item, err := getExpensePostgres(ctx, tripID, expenseID)
	if err != nil {
		return expense{}, err
	}

	if in.Category != nil {
		item.Category = strings.TrimSpace(*in.Category)
	}
	if in.Amount != nil {
		item.Amount = *in.Amount
	}
	if in.Currency != nil {
		item.Currency = strings.ToUpper(strings.TrimSpace(*in.Currency))
	}
	if in.ExpenseAt != nil {
		item.ExpenseAt = in.ExpenseAt
	}
	if in.Note != nil {
		item.Note = strings.TrimSpace(*in.Note)
	}

	var expenseAt *time.Time
	if item.ExpenseAt != nil {
		expenseAt, err = parseExpenseTime(*item.ExpenseAt)
		if err != nil {
			return expense{}, err
		}
	}

	var linkedItem any
	if item.LinkedItemID != nil && strings.TrimSpace(*item.LinkedItemID) != "" {
		linkedItem = strings.TrimSpace(*item.LinkedItemID)
	}

	res, err := p.Exec(ctx, `
		UPDATE expenses
		SET category = $3,
		    amount = $4,
		    currency = $5,
		    expense_at = $6,
		    note = NULLIF($7, ''),
		    linked_item_id = $8::uuid,
		    updated_at = now()
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, tripID, expenseID, item.Category, item.Amount, item.Currency, expenseAt, item.Note, linkedItem)
	if err != nil {
		return expense{}, err
	}
	if res.RowsAffected() == 0 {
		return expense{}, ErrExpenseNotFound
	}

	expenseVersionByID[expenseID] = expenseVersionByID[expenseID] + 1
	if expenseVersionByID[expenseID] == 0 {
		expenseVersionByID[expenseID] = 2
	}
	item.Version = expenseVersionByID[expenseID]
	return item, nil
}

func deleteExpensePostgres(ctx context.Context, tripID, expenseID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres budget store not configured")
	}

	res, err := p.Exec(ctx, `
		UPDATE expenses
		SET deleted_at = now(),
		    updated_at = now()
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, tripID, expenseID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrExpenseNotFound
	}
	delete(expenseVersionByID, expenseID)
	return nil
}

func ensureBudgetUser(ctx context.Context, p *pgxpool.Pool, userID string) error {
	email := "system@time-tree.local"
	if userID != defaultBudgetUserID {
		email = "user-" + userID + "@time-tree.local"
	}
	_, err := p.Exec(ctx, `
		INSERT INTO users (id, email, display_name, locale, timezone, default_currency, created_at, updated_at)
		VALUES ($1::uuid, $2, $3, 'zh-TW', 'Asia/Taipei', 'TWD', now(), now())
		ON CONFLICT (id) DO NOTHING
	`, userID, email, "System")
	return err
}

func parseExpenseTime(v string) (*time.Time, error) {
	value := strings.TrimSpace(v)
	if value == "" {
		return nil, nil
	}
	if t, err := time.Parse(time.RFC3339, value); err == nil {
		utc := t.UTC()
		return &utc, nil
	}
	if t, err := time.Parse("2006-01-02", value); err == nil {
		utc := t.UTC()
		return &utc, nil
	}
	return nil, errors.New("expenseAt must be RFC3339 or YYYY-MM-DD")
}
