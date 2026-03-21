-- Phase 1: Trips (with FK to users)
CREATE TABLE trips (
    id              UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    owner_user_id   UUID NOT NULL REFERENCES users(id),
    name            TEXT NOT NULL,
    destination_text TEXT,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    timezone        TEXT NOT NULL,
    currency        CHAR(3) NOT NULL,
    travelers_count INT NOT NULL CHECK (travelers_count > 0),
    status          TEXT NOT NULL DEFAULT 'draft',
    version         INT NOT NULL DEFAULT 1,
    tags            JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at     TIMESTAMPTZ,

    CONSTRAINT chk_trips_dates CHECK (end_date >= start_date),
    CONSTRAINT chk_trips_status CHECK (status IN ('draft', 'active', 'archived'))
);

CREATE INDEX idx_trips_owner_user_id ON trips(owner_user_id);
CREATE INDEX idx_trips_status ON trips(status);

-- ────────────────────────────────────────────────────────

CREATE TABLE trip_memberships (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id     UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        TEXT NOT NULL DEFAULT 'viewer',
    status      TEXT NOT NULL DEFAULT 'active',
    joined_at   TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_trip_memberships UNIQUE (trip_id, user_id),
    CONSTRAINT chk_membership_role CHECK (role IN ('owner', 'editor', 'commenter', 'viewer')),
    CONSTRAINT chk_membership_status CHECK (status IN ('active', 'removed'))
);

CREATE INDEX idx_trip_memberships_user_id ON trip_memberships(user_id);

-- ────────────────────────────────────────────────────────

CREATE TABLE trip_invitations (
    id                  UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    trip_id             UUID NOT NULL REFERENCES trips(id) ON DELETE CASCADE,
    invited_by_user_id  UUID NOT NULL REFERENCES users(id),
    invitee_email       CITEXT NOT NULL,
    role                TEXT NOT NULL DEFAULT 'viewer',
    token_hash          TEXT NOT NULL UNIQUE,
    status              TEXT NOT NULL DEFAULT 'pending',
    expires_at          TIMESTAMPTZ NOT NULL,
    accepted_at         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_invitation_role CHECK (role IN ('editor', 'commenter', 'viewer')),
    CONSTRAINT chk_invitation_status CHECK (status IN ('pending', 'accepted', 'revoked', 'expired'))
);

CREATE INDEX idx_trip_invitations_trip_id ON trip_invitations(trip_id);
CREATE INDEX idx_trip_invitations_invitee_email ON trip_invitations(invitee_email);
CREATE INDEX idx_trip_invitations_token_hash ON trip_invitations(token_hash);
