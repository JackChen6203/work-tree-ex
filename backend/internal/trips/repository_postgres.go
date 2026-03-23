package trips

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const defaultOwnerUserID = "00000000-0000-0000-0000-000000000001"

type postgresRepository struct {
	pool *pgxpool.Pool
}

func NewPostgresRepository(pool *pgxpool.Pool) Repository {
	return &postgresRepository{pool: pool}
}

func (r *postgresRepository) List(ctx context.Context) ([]trip, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, name, destination_text, start_date, end_date, timezone, currency,
		       travelers_count, status, version, created_at, updated_at
		FROM trips
		ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]trip, 0)
	for rows.Next() {
		t, err := scanTrip(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, t)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *postgresRepository) Create(ctx context.Context, in tripCreateInput, idempotencyKey string) (trip, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return trip{}, err
	}
	defer rollbackTx(ctx, tx)

	var existingTripID string
	err = tx.QueryRow(ctx, `SELECT trip_id::text FROM trip_idempotency_keys WHERE idempotency_key = $1`, idempotencyKey).Scan(&existingTripID)
	if err == nil {
		t, getErr := r.getByIDTx(ctx, tx, existingTripID)
		if getErr != nil {
			return trip{}, getErr
		}
		if err := tx.Commit(ctx); err != nil {
			return trip{}, err
		}
		return t, nil
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return trip{}, err
	}

	tripID := uuid.NewString()
	now := time.Now().UTC()
	_, err = tx.Exec(ctx, `
		INSERT INTO trips (
			id, owner_user_id, name, destination_text, start_date, end_date,
			timezone, currency, travelers_count, status, version, tags, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5::date, $6::date, $7, $8, $9, 'draft', 1, '[]'::jsonb, $10, $10)
	`, tripID, defaultOwnerUserID, in.Name, in.Destination, in.StartDate, in.EndDate, in.Timezone, in.Currency, in.Travelers, now)
	if err != nil {
		return trip{}, err
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO trip_idempotency_keys (idempotency_key, trip_id, created_at)
		VALUES ($1, $2, $3)
	`, idempotencyKey, tripID, now)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			t, getErr := r.getByIDTx(ctx, tx, tripID)
			if getErr != nil {
				return trip{}, getErr
			}
			if commitErr := tx.Commit(ctx); commitErr != nil {
				return trip{}, commitErr
			}
			return t, nil
		}
		return trip{}, err
	}

	t, err := r.getByIDTx(ctx, tx, tripID)
	if err != nil {
		return trip{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return trip{}, err
	}
	return t, nil
}

func (r *postgresRepository) Get(ctx context.Context, tripID string) (trip, error) {
	row := r.pool.QueryRow(ctx, `
		SELECT id, name, destination_text, start_date, end_date, timezone, currency,
		       travelers_count, status, version, created_at, updated_at
		FROM trips
		WHERE id = $1
	`, tripID)
	return scanTrip(row)
}

func (r *postgresRepository) Update(ctx context.Context, tripID string, expectedVersion int, in tripPatchInput) (trip, error) {
	tx, err := r.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return trip{}, err
	}
	defer rollbackTx(ctx, tx)

	current, err := r.getByIDTxForUpdate(ctx, tx, tripID)
	if err != nil {
		return trip{}, err
	}

	if current.Version != expectedVersion {
		return trip{}, ErrVersionConflict
	}

	if in.Name != nil {
		current.Name = *in.Name
	}
	if in.Destination != nil {
		current.Destination = *in.Destination
	}
	if in.StartDate != nil {
		current.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		current.EndDate = *in.EndDate
	}
	if in.Timezone != nil {
		current.Timezone = *in.Timezone
	}
	if in.Currency != nil {
		current.Currency = *in.Currency
	}
	if in.Travelers != nil && *in.Travelers > 0 {
		current.Travelers = *in.Travelers
	}
	if in.Status != nil {
		current.Status = *in.Status
	}

	now := time.Now().UTC()
	row := tx.QueryRow(ctx, `
		UPDATE trips
		SET name = $2,
			destination_text = $3,
			start_date = $4::date,
			end_date = $5::date,
			timezone = $6,
			currency = $7,
			travelers_count = $8,
			status = $9,
			version = version + 1,
			updated_at = $10
		WHERE id = $1
		RETURNING id, name, destination_text, start_date, end_date, timezone, currency,
		          travelers_count, status, version, created_at, updated_at
	`, tripID, current.Name, current.Destination, current.StartDate, current.EndDate, current.Timezone, current.Currency, current.Travelers, current.Status, now)

	updated, err := scanTrip(row)
	if err != nil {
		return trip{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return trip{}, err
	}
	return updated, nil
}

func (r *postgresRepository) getByIDTx(ctx context.Context, tx pgx.Tx, tripID string) (trip, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, name, destination_text, start_date, end_date, timezone, currency,
		       travelers_count, status, version, created_at, updated_at
		FROM trips
		WHERE id = $1
	`, tripID)
	return scanTrip(row)
}

func (r *postgresRepository) getByIDTxForUpdate(ctx context.Context, tx pgx.Tx, tripID string) (trip, error) {
	row := tx.QueryRow(ctx, `
		SELECT id, name, destination_text, start_date, end_date, timezone, currency,
		       travelers_count, status, version, created_at, updated_at
		FROM trips
		WHERE id = $1
		FOR UPDATE
	`, tripID)
	return scanTrip(row)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanTrip(scanner rowScanner) (trip, error) {
	var (
		t         trip
		startDate time.Time
		endDate   time.Time
	)
	err := scanner.Scan(
		&t.ID,
		&t.Name,
		&t.Destination,
		&startDate,
		&endDate,
		&t.Timezone,
		&t.Currency,
		&t.Travelers,
		&t.Status,
		&t.Version,
		&t.CreatedAt,
		&t.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return trip{}, ErrTripNotFound
		}
		return trip{}, err
	}

	t.StartDate = startDate.Format("2006-01-02")
	t.EndDate = endDate.Format("2006-01-02")
	return t, nil
}

func rollbackTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
