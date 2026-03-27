package itinerary

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	platformdb "github.com/solidityDeveloper/time_tree_ex/backend/internal/platform/database"
)

var (
	ErrItineraryTripNotFound    = errors.New("trip not found")
	ErrItineraryDayNotFound     = errors.New("itinerary day not found")
	ErrItineraryItemNotFound    = errors.New("itinerary item not found")
	ErrItineraryVersionConflict = errors.New("itinerary version conflict")
)

type itineraryValidationError struct {
	message string
}

func (e itineraryValidationError) Error() string {
	return e.message
}

func isValidationError(err error) bool {
	var validationErr itineraryValidationError
	return errors.As(err, &validationErr)
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

func listDaysPostgres(ctx context.Context, tripID string) ([]itineraryDay, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres itinerary store not configured")
	}

	if err := ensureTripDaysPostgres(ctx, p, tripID); err != nil {
		return nil, err
	}

	return fetchDaysPostgres(ctx, p, tripID)
}

func createItemPostgres(ctx context.Context, tripID string, in itemCreateInput) (itineraryItem, []string, error) {
	p := getPool()
	if p == nil {
		return itineraryItem{}, nil, errors.New("postgres itinerary store not configured")
	}

	normalizedTripID, err := normalizeUUIDField("tripId", tripID)
	if err != nil {
		return itineraryItem{}, nil, err
	}
	dayID, err := normalizeUUIDField("dayId", in.DayID)
	if err != nil {
		return itineraryItem{}, nil, err
	}

	startAt, err := parseOptionalRFC3339(in.StartAt, "startAt")
	if err != nil {
		return itineraryItem{}, nil, err
	}
	endAt, err := parseOptionalRFC3339(in.EndAt, "endAt")
	if err != nil {
		return itineraryItem{}, nil, err
	}

	placeSnapshotID, err := normalizeOptionalUUIDField(in.PlaceSnapshotID, "placeSnapshotId")
	if err != nil {
		return itineraryItem{}, nil, err
	}
	routeSnapshotID, err := normalizeOptionalUUIDField(in.RouteSnapshotID, "routeSnapshotId")
	if err != nil {
		return itineraryItem{}, nil, err
	}

	for attempt := 1; ; attempt++ {
		item, err := func() (itineraryItem, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return itineraryItem{}, err
			}
			defer rollbackItineraryTx(ctx, tx)

			if err := ensureTripDaysTx(ctx, tx, normalizedTripID); err != nil {
				return itineraryItem{}, err
			}
			exists, err := dayExistsTx(ctx, tx, normalizedTripID, dayID)
			if err != nil {
				return itineraryItem{}, err
			}
			if !exists {
				return itineraryItem{}, ErrItineraryDayNotFound
			}

			nextSort, err := nextSortOrderTx(ctx, tx, normalizedTripID, dayID)
			if err != nil {
				return itineraryItem{}, err
			}

			var item itineraryItem
			err = scanItineraryItem(tx.QueryRow(ctx, `
				INSERT INTO itinerary_items (
					trip_id, day_id, title, item_type, start_at, end_at, all_day, sort_order,
					note, provider_place_id, lat, lng, place_snapshot_id, route_snapshot_id,
					estimated_cost_amount, estimated_cost_currency, source_type, version, created_at, updated_at
				) VALUES (
					$1::uuid, $2::uuid, $3, $4, $5, $6, $7, $8,
					$9, $10, $11, $12, $13::uuid, $14::uuid,
					NULL, NULL, 'manual', 1, now(), now()
				)
				RETURNING id::text, day_id::text, title, item_type, start_at, end_at, all_day, sort_order,
				          note, provider_place_id, lat::float8, lng::float8, place_snapshot_id::text,
				          route_snapshot_id::text, estimated_cost_amount::float8, estimated_cost_currency, version
			`,
				normalizedTripID,
				dayID,
				strings.TrimSpace(in.Title),
				strings.TrimSpace(in.ItemType),
				startAt,
				endAt,
				in.AllDay,
				nextSort,
				nullableText(in.Note),
				nullableText(in.PlaceID),
				in.Lat,
				in.Lng,
				nullableUUIDArg(placeSnapshotID),
				nullableUUIDArg(routeSnapshotID),
			), &item)
			if err != nil {
				return itineraryItem{}, err
			}

			if err := tx.Commit(ctx); err != nil {
				return itineraryItem{}, err
			}
			return item, nil
		}()
		if err == nil {
			warnings, warningErr := detectTimeOverlapsPostgres(ctx, normalizedTripID, item.DayID)
			if warningErr != nil {
				return itineraryItem{}, nil, platformdb.WrapError(warningErr)
			}
			return item, warnings, nil
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return itineraryItem{}, nil, platformdb.DeadlockRetryExhaustedError(err)
		}
		return itineraryItem{}, nil, err
	}
}

func getItemPostgres(ctx context.Context, tripID, itemID string) (itineraryItem, error) {
	p := getPool()
	if p == nil {
		return itineraryItem{}, errors.New("postgres itinerary store not configured")
	}

	tripID, err := normalizeUUIDField("tripId", tripID)
	if err != nil {
		return itineraryItem{}, err
	}
	itemID, err = normalizeUUIDField("itemId", itemID)
	if err != nil {
		return itineraryItem{}, err
	}

	var item itineraryItem
	err = scanItineraryItem(p.QueryRow(ctx, `
		SELECT id::text, day_id::text, title, item_type, start_at, end_at, all_day, sort_order,
		       note, provider_place_id, lat::float8, lng::float8, place_snapshot_id::text,
		       route_snapshot_id::text, estimated_cost_amount::float8, estimated_cost_currency, version
		FROM itinerary_items
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, tripID, itemID), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return itineraryItem{}, ErrItineraryItemNotFound
	}
	if err != nil {
		return itineraryItem{}, err
	}

	return item, nil
}

func patchItemPostgres(ctx context.Context, tripID, itemID string, expectedVersion int, in itemPatchInput) (itineraryItem, error) {
	p := getPool()
	if p == nil {
		return itineraryItem{}, errors.New("postgres itinerary store not configured")
	}

	normalizedTripID, err := normalizeUUIDField("tripId", tripID)
	if err != nil {
		return itineraryItem{}, err
	}
	normalizedItemID, err := normalizeUUIDField("itemId", itemID)
	if err != nil {
		return itineraryItem{}, err
	}

	for attempt := 1; ; attempt++ {
		updated, err := func() (itineraryItem, error) {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return itineraryItem{}, err
			}
			defer rollbackItineraryTx(ctx, tx)

			if err := ensureTripDaysTx(ctx, tx, normalizedTripID); err != nil {
				return itineraryItem{}, err
			}

			current, err := getItemForUpdateTx(ctx, tx, normalizedTripID, normalizedItemID)
			if err != nil {
				return itineraryItem{}, err
			}
			if current.Version != expectedVersion {
				return itineraryItem{}, ErrItineraryVersionConflict
			}

			if in.Title != nil {
				current.Title = strings.TrimSpace(*in.Title)
			}

			if in.StartAt != nil {
				startAt, parseErr := parseOptionalRFC3339(in.StartAt, "startAt")
				if parseErr != nil {
					return itineraryItem{}, parseErr
				}
				current.StartAt = toRFC3339Ptr(startAt)
			}
			if in.EndAt != nil {
				endAt, parseErr := parseOptionalRFC3339(in.EndAt, "endAt")
				if parseErr != nil {
					return itineraryItem{}, parseErr
				}
				current.EndAt = toRFC3339Ptr(endAt)
			}
			if in.AllDay != nil {
				current.AllDay = *in.AllDay
			}
			if in.Note != nil {
				current.Note = cloneStringPtr(in.Note)
			}
			if in.PlaceID != nil {
				current.PlaceID = cloneStringPtr(in.PlaceID)
			}
			if in.Lat != nil {
				current.Lat = in.Lat
			}
			if in.Lng != nil {
				current.Lng = in.Lng
			}
			if in.PlaceSnapshotID != nil {
				normalized, normalizeErr := normalizeOptionalUUIDField(in.PlaceSnapshotID, "placeSnapshotId")
				if normalizeErr != nil {
					return itineraryItem{}, normalizeErr
				}
				current.PlaceSnapshotID = normalized
			}
			if in.RouteSnapshotID != nil {
				normalized, normalizeErr := normalizeOptionalUUIDField(in.RouteSnapshotID, "routeSnapshotId")
				if normalizeErr != nil {
					return itineraryItem{}, normalizeErr
				}
				current.RouteSnapshotID = normalized
			}
			if in.SortOrder != nil {
				current.SortOrder = *in.SortOrder
			}

			if in.DayID != nil {
				targetDayID, normalizeErr := normalizeUUIDField("dayId", *in.DayID)
				if normalizeErr != nil {
					return itineraryItem{}, normalizeErr
				}
				if targetDayID != current.DayID {
					exists, existsErr := dayExistsTx(ctx, tx, normalizedTripID, targetDayID)
					if existsErr != nil {
						return itineraryItem{}, existsErr
					}
					if !exists {
						return itineraryItem{}, ErrItineraryDayNotFound
					}
					current.DayID = targetDayID
					if in.SortOrder == nil {
						nextSort, nextErr := nextSortOrderTx(ctx, tx, normalizedTripID, targetDayID)
						if nextErr != nil {
							return itineraryItem{}, nextErr
						}
						current.SortOrder = nextSort
					}
				}
			}

			startAt, err := parseOptionalRFC3339(current.StartAt, "startAt")
			if err != nil {
				return itineraryItem{}, err
			}
			endAt, err := parseOptionalRFC3339(current.EndAt, "endAt")
			if err != nil {
				return itineraryItem{}, err
			}

			var updated itineraryItem
			err = scanItineraryItem(tx.QueryRow(ctx, `
				UPDATE itinerary_items
				SET day_id = $3::uuid,
					title = $4,
					start_at = $5,
					end_at = $6,
					all_day = $7,
					sort_order = $8,
					note = $9,
					provider_place_id = $10,
					lat = $11,
					lng = $12,
					place_snapshot_id = $13::uuid,
					route_snapshot_id = $14::uuid,
					version = version + 1,
					updated_at = now()
				WHERE id = $1::uuid
				  AND trip_id = $2::uuid
				  AND deleted_at IS NULL
				  AND version = $15
				RETURNING id::text, day_id::text, title, item_type, start_at, end_at, all_day, sort_order,
				          note, provider_place_id, lat::float8, lng::float8, place_snapshot_id::text,
				          route_snapshot_id::text, estimated_cost_amount::float8, estimated_cost_currency, version
			`,
				normalizedItemID,
				normalizedTripID,
				current.DayID,
				current.Title,
				startAt,
				endAt,
				current.AllDay,
				current.SortOrder,
				nullableText(current.Note),
				nullableText(current.PlaceID),
				current.Lat,
				current.Lng,
				nullableUUIDArg(current.PlaceSnapshotID),
				nullableUUIDArg(current.RouteSnapshotID),
				expectedVersion,
			), &updated)
			if errors.Is(err, pgx.ErrNoRows) {
				return itineraryItem{}, ErrItineraryVersionConflict
			}
			if err != nil {
				return itineraryItem{}, err
			}

			if err := tx.Commit(ctx); err != nil {
				return itineraryItem{}, err
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
			return itineraryItem{}, platformdb.DeadlockRetryExhaustedError(err)
		}
		return itineraryItem{}, err
	}
}

func deleteItemPostgres(ctx context.Context, tripID, itemID string) error {
	p := getPool()
	if p == nil {
		return errors.New("postgres itinerary store not configured")
	}

	tripID, err := normalizeUUIDField("tripId", tripID)
	if err != nil {
		return err
	}
	itemID, err = normalizeUUIDField("itemId", itemID)
	if err != nil {
		return err
	}

	res, err := p.Exec(ctx, `
		UPDATE itinerary_items
		SET deleted_at = now(), updated_at = now()
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
	`, tripID, itemID)
	if err != nil {
		return err
	}
	if res.RowsAffected() == 0 {
		return ErrItineraryItemNotFound
	}
	return nil
}

func reorderItemsPostgres(ctx context.Context, tripID string, in reorderInput) ([]itineraryDay, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres itinerary store not configured")
	}

	normalizedTripID, err := normalizeUUIDField("tripId", tripID)
	if err != nil {
		return nil, err
	}

	for attempt := 1; ; attempt++ {
		err := func() error {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return err
			}
			defer rollbackItineraryTx(ctx, tx)

			if err := ensureTripDaysTx(ctx, tx, normalizedTripID); err != nil {
				return err
			}

			dayIDs, err := listDayIDsTx(ctx, tx, normalizedTripID)
			if err != nil {
				return err
			}

			for _, op := range in.Operations {
				targetDayID, normalizeErr := normalizeUUIDField("targetDayId", op.TargetDayID)
				if normalizeErr != nil {
					return normalizeErr
				}
				if _, ok := dayIDs[targetDayID]; !ok {
					return ErrItineraryDayNotFound
				}

				itemID, normalizeErr := normalizeUUIDField("itemId", op.ItemID)
				if normalizeErr != nil {
					return normalizeErr
				}

				res, execErr := tx.Exec(ctx, `
					UPDATE itinerary_items
					SET day_id = $3::uuid,
					    sort_order = $4,
					    version = version + 1,
					    updated_at = now()
					WHERE trip_id = $1::uuid
					  AND id = $2::uuid
					  AND deleted_at IS NULL
				`, normalizedTripID, itemID, targetDayID, op.TargetSortOrder)
				if execErr != nil {
					return execErr
				}
				if res.RowsAffected() == 0 {
					return ErrItineraryItemNotFound
				}
			}

			return tx.Commit(ctx)
		}()
		if err == nil {
			return fetchDaysPostgres(ctx, p, normalizedTripID)
		}

		err = platformdb.WrapError(err)
		if platformdb.ShouldRetryDeadlock(err, attempt) {
			time.Sleep(platformdb.DeadlockRetryDelay(attempt))
			continue
		}
		if platformdb.IsDeadlock(err) {
			return nil, platformdb.DeadlockRetryExhaustedError(err)
		}
		return nil, err
	}
}

func fetchDaysPostgres(ctx context.Context, q dbQuerier, tripID string) ([]itineraryDay, error) {
	dayRows, err := q.Query(ctx, `
		SELECT id::text, trip_date, sort_order
		FROM itinerary_days
		WHERE trip_id = $1::uuid
		ORDER BY sort_order ASC, day_index ASC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer dayRows.Close()

	days := make([]itineraryDay, 0)
	dayIndex := make(map[string]int)

	for dayRows.Next() {
		var (
			dayID     string
			tripDate  time.Time
			sortOrder int
		)
		if err := dayRows.Scan(&dayID, &tripDate, &sortOrder); err != nil {
			return nil, err
		}
		dayIndex[dayID] = len(days)
		days = append(days, itineraryDay{
			DayID:     dayID,
			Date:      tripDate.Format("2006-01-02"),
			SortOrder: sortOrder,
			Items:     []itineraryItem{},
		})
	}
	if err := dayRows.Err(); err != nil {
		return nil, err
	}

	if len(days) == 0 {
		return []itineraryDay{}, nil
	}

	itemRows, err := q.Query(ctx, `
		SELECT id::text, day_id::text, title, item_type, start_at, end_at, all_day, sort_order,
		       note, provider_place_id, lat::float8, lng::float8, place_snapshot_id::text,
		       route_snapshot_id::text, estimated_cost_amount::float8, estimated_cost_currency, version
		FROM itinerary_items
		WHERE trip_id = $1::uuid
		  AND deleted_at IS NULL
		ORDER BY day_id ASC, sort_order ASC, created_at ASC
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer itemRows.Close()

	for itemRows.Next() {
		var item itineraryItem
		if err := scanItineraryItem(itemRows, &item); err != nil {
			return nil, err
		}
		idx, ok := dayIndex[item.DayID]
		if !ok {
			continue
		}
		days[idx].Items = append(days[idx].Items, item)
	}
	if err := itemRows.Err(); err != nil {
		return nil, err
	}

	for i := range days {
		sort.SliceStable(days[i].Items, func(a, b int) bool {
			return days[i].Items[a].SortOrder < days[i].Items[b].SortOrder
		})
	}

	return days, nil
}

func detectTimeOverlapsPostgres(ctx context.Context, tripID, dayID string) ([]string, error) {
	p := getPool()
	if p == nil {
		return nil, errors.New("postgres itinerary store not configured")
	}

	rows, err := p.Query(ctx, `
		SELECT title, start_at, end_at
		FROM itinerary_items
		WHERE trip_id = $1::uuid
		  AND day_id = $2::uuid
		  AND deleted_at IS NULL
		  AND start_at IS NOT NULL
		  AND end_at IS NOT NULL
	`, tripID, dayID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type timedItem struct {
		title string
		start time.Time
		end   time.Time
	}

	items := make([]timedItem, 0)
	for rows.Next() {
		var item timedItem
		if err := rows.Scan(&item.title, &item.start, &item.end); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	warnings := make([]string, 0)
	for i := 0; i < len(items); i++ {
		for j := i + 1; j < len(items); j++ {
			a := items[i]
			b := items[j]
			if a.start.Before(b.end) && b.start.Before(a.end) {
				warnings = append(warnings, "time overlap between '"+a.title+"' and '"+b.title+"'")
			}
		}
	}
	return warnings, nil
}

func ensureTripDaysPostgres(ctx context.Context, p *pgxpool.Pool, tripID string) error {
	for attempt := 1; ; attempt++ {
		err := func() error {
			tx, err := p.BeginTx(ctx, pgx.TxOptions{})
			if err != nil {
				return err
			}
			defer rollbackItineraryTx(ctx, tx)

			if err := ensureTripDaysTx(ctx, tx, tripID); err != nil {
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

func ensureTripDaysTx(ctx context.Context, tx pgx.Tx, tripID string) error {
	tripID, err := normalizeUUIDField("tripId", tripID)
	if err != nil {
		return err
	}

	var (
		startDate time.Time
		endDate   time.Time
	)
	err = tx.QueryRow(ctx, `
		SELECT start_date, end_date
		FROM trips
		WHERE id = $1::uuid
	`, tripID).Scan(&startDate, &endDate)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrItineraryTripNotFound
	}
	if err != nil {
		return err
	}

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

func dayExistsTx(ctx context.Context, tx pgx.Tx, tripID, dayID string) (bool, error) {
	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS(
			SELECT 1
			FROM itinerary_days
			WHERE trip_id = $1::uuid
			  AND id = $2::uuid
		)
	`, tripID, dayID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}

func listDayIDsTx(ctx context.Context, tx pgx.Tx, tripID string) (map[string]struct{}, error) {
	rows, err := tx.Query(ctx, `
		SELECT id::text
		FROM itinerary_days
		WHERE trip_id = $1::uuid
	`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	ids := make(map[string]struct{})
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return ids, nil
}

func nextSortOrderTx(ctx context.Context, tx pgx.Tx, tripID, dayID string) (int, error) {
	var next int
	if err := tx.QueryRow(ctx, `
		SELECT COALESCE(MAX(sort_order), 0) + 1
		FROM itinerary_items
		WHERE trip_id = $1::uuid
		  AND day_id = $2::uuid
		  AND deleted_at IS NULL
	`, tripID, dayID).Scan(&next); err != nil {
		return 0, err
	}
	return next, nil
}

func getItemForUpdateTx(ctx context.Context, tx pgx.Tx, tripID, itemID string) (itineraryItem, error) {
	var item itineraryItem
	err := scanItineraryItem(tx.QueryRow(ctx, `
		SELECT id::text, day_id::text, title, item_type, start_at, end_at, all_day, sort_order,
		       note, provider_place_id, lat::float8, lng::float8, place_snapshot_id::text,
		       route_snapshot_id::text, estimated_cost_amount::float8, estimated_cost_currency, version
		FROM itinerary_items
		WHERE trip_id = $1::uuid
		  AND id = $2::uuid
		  AND deleted_at IS NULL
		FOR UPDATE
	`, tripID, itemID), &item)
	if errors.Is(err, pgx.ErrNoRows) {
		return itineraryItem{}, ErrItineraryItemNotFound
	}
	if err != nil {
		return itineraryItem{}, err
	}
	return item, nil
}

type dbQuerier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

type itineraryRowScanner interface {
	Scan(dest ...any) error
}

func scanItineraryItem(scanner itineraryRowScanner, item *itineraryItem) error {
	var (
		startAt               *time.Time
		endAt                 *time.Time
		note                  *string
		providerPlaceID       *string
		placeSnapshotID       *string
		routeSnapshotID       *string
		estimatedCostAmount   *float64
		estimatedCostCurrency *string
	)

	if err := scanner.Scan(
		&item.ID,
		&item.DayID,
		&item.Title,
		&item.ItemType,
		&startAt,
		&endAt,
		&item.AllDay,
		&item.SortOrder,
		&note,
		&providerPlaceID,
		&item.Lat,
		&item.Lng,
		&placeSnapshotID,
		&routeSnapshotID,
		&estimatedCostAmount,
		&estimatedCostCurrency,
		&item.Version,
	); err != nil {
		return err
	}

	item.StartAt = toRFC3339Ptr(startAt)
	item.EndAt = toRFC3339Ptr(endAt)
	item.Note = trimOptionalString(note)
	item.PlaceID = trimOptionalString(providerPlaceID)
	item.PlaceSnapshotID = trimOptionalString(placeSnapshotID)
	item.RouteSnapshotID = trimOptionalString(routeSnapshotID)
	item.EstimatedCostAmount = estimatedCostAmount
	item.EstimatedCostCurrency = trimOptionalString(estimatedCostCurrency)
	return nil
}

func nullableText(value *string) any {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func nullableUUIDArg(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func trimOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func parseOptionalRFC3339(value *string, fieldName string) (*time.Time, error) {
	if value == nil {
		return nil, nil
	}
	raw := strings.TrimSpace(*value)
	if raw == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return nil, itineraryValidationError{message: fieldName + " must be a valid ISO-8601 timestamp"}
	}
	parsed = parsed.UTC()
	return &parsed, nil
}

func normalizeUUIDField(fieldName, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", itineraryValidationError{message: fieldName + " is required"}
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return "", itineraryValidationError{message: fieldName + " must be a valid UUID"}
	}
	return trimmed, nil
}

func normalizeOptionalUUIDField(value *string, fieldName string) (*string, error) {
	if value == nil {
		return nil, nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil, nil
	}
	if _, err := uuid.Parse(trimmed); err != nil {
		return nil, itineraryValidationError{message: fieldName + " must be a valid UUID"}
	}
	return &trimmed, nil
}

func cloneStringPtr(value *string) *string {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func toRFC3339Ptr(value *time.Time) *string {
	if value == nil {
		return nil
	}
	formatted := value.UTC().Format(time.RFC3339)
	return &formatted
}

func rollbackItineraryTx(ctx context.Context, tx pgx.Tx) {
	_ = tx.Rollback(ctx)
}
