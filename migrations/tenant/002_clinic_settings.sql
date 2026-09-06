-- Clinic settings extension. All columns are additive and nullable-safe so
-- this migration can be re-applied against existing tenant databases.

ALTER TABLE clinics
    ADD COLUMN IF NOT EXISTS logo_url        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS address         TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS locality        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS pin_code        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS state           TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS country         TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phones          JSONB   NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS email           TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS website         TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS gstin           TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS facility_id     TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS time_format     TEXT    NOT NULL DEFAULT '12h',
    ADD COLUMN IF NOT EXISTS system_language TEXT    NOT NULL DEFAULT 'en',
    ADD COLUMN IF NOT EXISTS time_zone       TEXT    NOT NULL DEFAULT 'UTC',
    ADD COLUMN IF NOT EXISTS date_format     TEXT    NOT NULL DEFAULT 'MM/DD/YYYY',
    ADD COLUMN IF NOT EXISTS currency        TEXT    NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS timings         JSONB   NOT NULL DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS is_primary      BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS is_archived     BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS archived_at     TIMESTAMPTZ;

-- At most one primary clinic per tenant.
CREATE UNIQUE INDEX IF NOT EXISTS clinics_one_primary_idx
    ON clinics ((1)) WHERE is_primary;
