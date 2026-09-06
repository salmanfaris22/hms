-- Super admin panel: platform owners who manage hospital organisations.
-- Also extends the `tenants` table with subscription + quota + module controls.

CREATE TABLE IF NOT EXISTS super_admins (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         TEXT NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    full_name     TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS max_clinics        INT      NOT NULL DEFAULT 5,
    ADD COLUMN IF NOT EXISTS storage_quota_gb   INT      NOT NULL DEFAULT 10,
    ADD COLUMN IF NOT EXISTS modules            TEXT[]   NOT NULL DEFAULT ARRAY[
        'dashboard','appointments','patients','laboratory',
        'inventory','billing','finance','staff','reports','settings'
    ],
    ADD COLUMN IF NOT EXISTS subscription_start DATE,
    ADD COLUMN IF NOT EXISTS subscription_end   DATE,
    ADD COLUMN IF NOT EXISTS is_active          BOOLEAN  NOT NULL DEFAULT true;
