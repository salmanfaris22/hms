-- The Add Vitals modal (2862:39200) records a batch of readings with one shared
-- clinical note, and its Configure dialog (3055:43827 / 3055:43826) chooses which
-- vitals and calculators a clinic uses. Neither had anywhere to live.
--
-- Every migration re-runs on boot, so both statements must be idempotent.
ALTER TABLE patient_vitals ADD COLUMN IF NOT EXISTS notes TEXT NOT NULL DEFAULT '';

CREATE TABLE IF NOT EXISTS patient_vitals_config (
    clinic_id  UUID PRIMARY KEY REFERENCES clinics(id) ON DELETE CASCADE,
    config     JSONB       NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
