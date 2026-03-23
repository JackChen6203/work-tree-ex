-- Phase 1: Itinerary Days and Items

CREATE TABLE itinerary_days (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id     UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    trip_date   DATE NOT NULL,
    day_index   INT NOT NULL,
    sort_order  INT NOT NULL,
    version     INT NOT NULL DEFAULT 1,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_itinerary_days_trip_date UNIQUE (trip_id, trip_date),
    CONSTRAINT uq_itinerary_days_sort UNIQUE (trip_id, sort_order) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_itinerary_days_trip_id ON itinerary_days(trip_id);

-- ────────────────────────────────────────────────────────

CREATE TABLE place_snapshots (
    id                      UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider                TEXT NOT NULL,
    provider_place_id       TEXT NOT NULL,
    name                    TEXT NOT NULL,
    address                 TEXT,
    lat                     NUMERIC(10,7) NOT NULL,
    lng                     NUMERIC(10,7) NOT NULL,
    categories              JSONB NOT NULL DEFAULT '[]',
    opening_hours           JSONB,
    raw_normalized_payload  JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_place_snapshots_provider ON place_snapshots(provider, provider_place_id);

-- ────────────────────────────────────────────────────────

CREATE TABLE route_snapshots (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    provider                 TEXT NOT NULL,
    mode                     TEXT NOT NULL,
    distance_meters          INT NOT NULL,
    duration_seconds         INT NOT NULL,
    estimated_cost_amount    NUMERIC(14,2),
    estimated_cost_currency  CHAR(3),
    raw_normalized_payload   JSONB,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ────────────────────────────────────────────────────────

CREATE TABLE itinerary_items (
    id                       UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id                  UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    day_id                   UUID NOT NULL REFERENCES itinerary_days(id) ON DELETE CASCADE,
    title                    TEXT NOT NULL,
    item_type                TEXT NOT NULL,
    start_at                 TIMESTAMPTZ,
    end_at                   TIMESTAMPTZ,
    all_day                  BOOLEAN NOT NULL DEFAULT false,
    sort_order               INT NOT NULL,
    note                     TEXT,
    provider_place_id        TEXT,
    lat                      NUMERIC(10,7),
    lng                      NUMERIC(10,7),
    place_snapshot_id        UUID REFERENCES place_snapshots(id),
    route_snapshot_id        UUID REFERENCES route_snapshots(id),
    estimated_cost_amount    NUMERIC(14,2),
    estimated_cost_currency  TEXT,
    source_type              TEXT,
    source_draft_id          UUID,
    version                  INT NOT NULL DEFAULT 1,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at               TIMESTAMPTZ,

    CONSTRAINT chk_item_type CHECK (item_type IN ('place_visit', 'meal', 'transit', 'hotel', 'free_time', 'custom')),
    CONSTRAINT chk_item_time CHECK (end_at IS NULL OR start_at IS NULL OR end_at >= start_at)
);

CREATE INDEX idx_itinerary_items_trip_id ON itinerary_items(trip_id);
CREATE INDEX idx_itinerary_items_day_id ON itinerary_items(day_id);
CREATE INDEX idx_itinerary_items_deleted_at ON itinerary_items(deleted_at) WHERE deleted_at IS NULL;
