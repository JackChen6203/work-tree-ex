-- Phase 3: AI Planning

CREATE TABLE ai_plan_requests (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id               UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    requested_by_user_id  UUID NOT NULL REFERENCES users(id),
    provider_config_id    UUID NOT NULL REFERENCES llm_provider_configs(id),
    status                TEXT NOT NULL DEFAULT 'queued',
    prompt_context        JSONB,
    prompt_tokens         INT,
    completion_tokens     INT,
    estimated_cost        NUMERIC(10,6),
    queued_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    started_at            TIMESTAMPTZ,
    finished_at           TIMESTAMPTZ,
    failure_code          TEXT,
    failure_message       TEXT,

    CONSTRAINT chk_ai_request_status CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled'))
);

CREATE INDEX idx_ai_plan_requests_trip_id ON ai_plan_requests(trip_id);
CREATE INDEX idx_ai_plan_requests_status ON ai_plan_requests(status);

-- ────────────────────────────────────────────────────────

CREATE TABLE ai_plan_drafts (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id         UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    request_id      UUID NOT NULL REFERENCES ai_plan_requests(id) ON DELETE CASCADE,
    title           TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'valid',
    draft_payload   JSONB NOT NULL,
    summary_payload JSONB,
    version         INT NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_ai_draft_status CHECK (status IN ('valid', 'warning', 'invalid'))
);

CREATE INDEX idx_ai_plan_drafts_trip_id ON ai_plan_drafts(trip_id);
CREATE INDEX idx_ai_plan_drafts_request_id ON ai_plan_drafts(request_id);

-- ────────────────────────────────────────────────────────

CREATE TABLE ai_plan_validation_results (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    draft_id    UUID NOT NULL REFERENCES ai_plan_drafts(id) ON DELETE CASCADE,
    severity    TEXT NOT NULL,
    rule_code   TEXT NOT NULL,
    message     TEXT NOT NULL,
    details     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_validation_severity CHECK (severity IN ('info', 'warning', 'error'))
);

CREATE INDEX idx_ai_validation_draft_id ON ai_plan_validation_results(draft_id);
