-- Fix pathology_parameters schema:
--   ref_min / ref_max were NUMERIC NOT NULL, which rejects empty or range-style
--   inputs ("12-17.5") and blocks the "Add Parameter" form entirely.
-- Switch to TEXT so the form's free-text values (single number, range, or
-- blank) round-trip losslessly. Add patient_groups JSONB for per-patient-group
-- ranges (Male / Female / Pregnant / Infant / etc.). Also self-heal any columns
-- that are missing from older partial-migration installs.

-- First ensure every expected column exists (earlier installs may be missing
-- some). All ADD COLUMN statements are idempotent.
ALTER TABLE pathology_parameters
    ADD COLUMN IF NOT EXISTS units          TEXT  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ref_min        TEXT  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS ref_max        TEXT  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS method         TEXT  NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS patient_groups JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN IF NOT EXISTS created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN IF NOT EXISTS updated_at     TIMESTAMPTZ NOT NULL DEFAULT now();

-- The create handler uses ON CONFLICT (clinic_id, lower(parameter_name)),
-- which requires a UNIQUE index. Migration 010 only created a plain index.
DROP INDEX IF EXISTS pathology_parameters_clinic_name_idx;
CREATE UNIQUE INDEX IF NOT EXISTS pathology_parameters_clinic_name_ux
    ON pathology_parameters (clinic_id, lower(parameter_name));

-- Convert ref_min / ref_max to TEXT if they are still numeric. Guarded by a
-- data-type check so repeated runs are cheap no-ops.
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'pathology_parameters'
          AND column_name = 'ref_min'
          AND data_type  = 'numeric'
    ) THEN
        ALTER TABLE pathology_parameters
            ALTER COLUMN ref_min DROP DEFAULT,
            ALTER COLUMN ref_min TYPE TEXT USING ref_min::text,
            ALTER COLUMN ref_min SET DEFAULT '';
    END IF;

    IF EXISTS (
        SELECT 1 FROM information_schema.columns
        WHERE table_name = 'pathology_parameters'
          AND column_name = 'ref_max'
          AND data_type  = 'numeric'
    ) THEN
        ALTER TABLE pathology_parameters
            ALTER COLUMN ref_max DROP DEFAULT,
            ALTER COLUMN ref_max TYPE TEXT USING ref_max::text,
            ALTER COLUMN ref_max SET DEFAULT '';
    END IF;
END $$;
