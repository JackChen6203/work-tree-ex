-- Phase 0: Users, Preferences, LLM Provider Configs

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email           CITEXT NOT NULL UNIQUE,
    display_name    TEXT NOT NULL,
    locale          TEXT NOT NULL DEFAULT 'zh-TW',
    timezone        TEXT NOT NULL DEFAULT 'Asia/Taipei',
    default_currency TEXT NOT NULL DEFAULT 'TWD',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at      TIMESTAMPTZ
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_deleted_at ON users(deleted_at) WHERE deleted_at IS NULL;

-- ────────────────────────────────────────────────────────

CREATE TABLE user_preferences (
    id                    UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id               UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    explicit_preferences  JSONB NOT NULL DEFAULT '{}',
    inferred_preferences  JSONB NOT NULL DEFAULT '{}',
    version               INT NOT NULL DEFAULT 1,
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_user_preferences_user UNIQUE (user_id)
);

-- ────────────────────────────────────────────────────────

CREATE TABLE llm_provider_configs (
    id                   UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id              UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    provider             TEXT NOT NULL,
    label                TEXT NOT NULL DEFAULT '',
    encrypted_key        TEXT NOT NULL,
    encrypted_key_kms_ref TEXT,
    base_url             TEXT,
    model                TEXT NOT NULL,
    is_active            BOOLEAN NOT NULL DEFAULT true,
    last_validated_at    TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_llm_provider_configs_user_id ON llm_provider_configs(user_id);
