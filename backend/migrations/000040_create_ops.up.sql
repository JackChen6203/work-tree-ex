-- Phase 4: Notifications, Share Links, Outbox Events, Audit Logs

CREATE TABLE notifications (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    trip_id     UUID REFERENCES trips(id) ON DELETE CASCADE,
    type        TEXT NOT NULL,
    title       TEXT NOT NULL,
    body        TEXT NOT NULL DEFAULT '',
    payload     JSONB NOT NULL DEFAULT '{}',
    read_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_notifications_user_id ON notifications(user_id);
CREATE INDEX idx_notifications_user_unread ON notifications(user_id, created_at DESC) WHERE read_at IS NULL;

-- ────────────────────────────────────────────────────────

CREATE TABLE share_links (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id       UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    token_hash    TEXT NOT NULL UNIQUE,
    access_scope  TEXT NOT NULL DEFAULT 'read',
    expires_at    TIMESTAMPTZ NOT NULL,
    revoked_at    TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_share_scope CHECK (access_scope IN ('read'))
);

CREATE INDEX idx_share_links_trip_id ON share_links(trip_id);
CREATE INDEX idx_share_links_token_hash ON share_links(token_hash);

-- ────────────────────────────────────────────────────────

CREATE TABLE outbox_events (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id         UUID REFERENCES trips(id) ON DELETE SET NULL,
    aggregate_type  TEXT NOT NULL,
    aggregate_id    TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    payload         JSONB NOT NULL,
    status          TEXT NOT NULL DEFAULT 'pending',
    retry_count     INT NOT NULL DEFAULT 0,
    available_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_outbox_status CHECK (status IN ('pending', 'processing', 'done', 'failed', 'dead'))
);

CREATE INDEX idx_outbox_events_status ON outbox_events(status, available_at) WHERE status IN ('pending', 'processing');

-- ────────────────────────────────────────────────────────

CREATE TABLE audit_logs (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    actor_user_id   UUID REFERENCES users(id),
    action          TEXT NOT NULL,
    resource_type   TEXT NOT NULL,
    resource_id     TEXT NOT NULL,
    before_state    JSONB,
    after_state     JSONB,
    request_id      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_actor ON audit_logs(actor_user_id);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at);
