package trips

import (
	"context"
	"errors"
)

var (
	ErrTripNotFound    = errors.New("trip not found")
	ErrVersionConflict = errors.New("trip version conflict")
)

type Repository interface {
	List(ctx context.Context) ([]trip, error)
	Create(ctx context.Context, in tripCreateInput, idempotencyKey string) (trip, error)
	Get(ctx context.Context, tripID string) (trip, error)
	Update(ctx context.Context, tripID string, expectedVersion int, in tripPatchInput) (trip, error)
}

var activeRepository Repository = newMemoryRepository()

func SetRepository(repo Repository) {
	if repo == nil {
		activeRepository = newMemoryRepository()
		return
	}
	activeRepository = repo
}
