-- Phase 2: Budget Profiles and Expenses

CREATE TABLE budget_profiles (
    id                UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id           UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    total_budget      NUMERIC(14,2),
    per_person_budget NUMERIC(14,2),
    per_day_budget    NUMERIC(14,2),
    currency          CHAR(3) NOT NULL,
    category_plan     JSONB NOT NULL DEFAULT '[]',
    version           INT NOT NULL DEFAULT 1,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_budget_profiles_trip UNIQUE (trip_id)
);

-- ────────────────────────────────────────────────────────

CREATE TABLE expenses (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id             UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    created_by_user_id  UUID NOT NULL REFERENCES users(id),
    category            TEXT NOT NULL,
    amount              NUMERIC(14,2) NOT NULL CHECK (amount >= 0),
    currency            CHAR(3) NOT NULL,
    expense_at          TIMESTAMPTZ,
    note                TEXT,
    linked_item_id      UUID REFERENCES itinerary_items(id) ON DELETE SET NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at          TIMESTAMPTZ,

    CONSTRAINT chk_expense_category CHECK (category IN ('lodging', 'transit', 'food', 'attraction', 'shopping', 'misc'))
);

CREATE INDEX idx_expenses_trip_id ON expenses(trip_id);
CREATE INDEX idx_expenses_created_by ON expenses(created_by_user_id);
CREATE INDEX idx_expenses_deleted_at ON expenses(deleted_at) WHERE deleted_at IS NULL;
