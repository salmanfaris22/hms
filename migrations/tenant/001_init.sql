-- Tenant database: each hospital gets its own database. No tenant_id column
-- is needed here — isolation is enforced at the database level.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'staff',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS clinics (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    clinic_type TEXT NOT NULL DEFAULT '',
    location    TEXT NOT NULL DEFAULT '',
    color       TEXT NOT NULL DEFAULT 'purple',
    invite_code TEXT NOT NULL UNIQUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS clinic_members (
    clinic_id        UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    user_id          UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role             TEXT NOT NULL DEFAULT 'admin',
    is_default       BOOLEAN NOT NULL DEFAULT false,
    joined_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_accessed_at TIMESTAMPTZ,
    PRIMARY KEY (clinic_id, user_id)
);
CREATE INDEX IF NOT EXISTS clinic_members_user_idx ON clinic_members (user_id);
