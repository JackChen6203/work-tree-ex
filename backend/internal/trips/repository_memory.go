package trips

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
)

type memoryRepository struct {
	mu          sync.RWMutex
	items       map[string]trip
	idempotency map[string]string
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{
		items:       make(map[string]trip),
		idempotency: make(map[string]string),
	}
}

func (r *memoryRepository) List(_ context.Context) ([]trip, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	items := make([]trip, 0, len(r.items))
	for _, t := range r.items {
		items = append(items, t)
	}
	return items, nil
}

func (r *memoryRepository) Create(_ context.Context, in tripCreateInput, idempotencyKey string) (trip, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if existingID, ok := r.idempotency[idempotencyKey]; ok {
		return r.items[existingID], nil
	}

	now := time.Now().UTC()
	t := trip{
		ID:          uuid.NewString(),
		Name:        in.Name,
		Destination: in.Destination,
		StartDate:   in.StartDate,
		EndDate:     in.EndDate,
		Timezone:    in.Timezone,
		Currency:    in.Currency,
		Travelers:   in.Travelers,
		Status:      "draft",
		Version:     1,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	r.items[t.ID] = t
	r.idempotency[idempotencyKey] = t.ID
	return t, nil
}

func (r *memoryRepository) Get(_ context.Context, tripID string) (trip, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	t, ok := r.items[tripID]
	if !ok {
		return trip{}, ErrTripNotFound
	}
	return t, nil
}

func (r *memoryRepository) Update(_ context.Context, tripID string, expectedVersion int, in tripPatchInput) (trip, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	t, ok := r.items[tripID]
	if !ok {
		return trip{}, ErrTripNotFound
	}

	if t.Version != expectedVersion {
		return trip{}, ErrVersionConflict
	}

	if in.Name != nil {
		t.Name = *in.Name
	}
	if in.Destination != nil {
		t.Destination = *in.Destination
	}
	if in.StartDate != nil {
		t.StartDate = *in.StartDate
	}
	if in.EndDate != nil {
		t.EndDate = *in.EndDate
	}
	if in.Timezone != nil {
		t.Timezone = *in.Timezone
	}
	if in.Currency != nil {
		t.Currency = *in.Currency
	}
	if in.Travelers != nil && *in.Travelers > 0 {
		t.Travelers = *in.Travelers
	}
	if in.Status != nil {
		t.Status = *in.Status
	}

	t.Version++
	t.UpdatedAt = time.Now().UTC()
	r.items[tripID] = t
	return t, nil
}
