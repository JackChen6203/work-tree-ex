CREATE TABLE trip_idempotency_keys (
    idempotency_key TEXT PRIMARY KEY,
    trip_id UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trip_idempotency_keys_trip_id ON trip_idempotency_keys(trip_id);
