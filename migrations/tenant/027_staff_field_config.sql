-- Per-clinic staff add-form field configuration.
-- Stored as JSONB: { "<fieldKey>": { "visible": bool, "required": bool } }
CREATE TABLE IF NOT EXISTS staff_field_config (
    clinic_id  UUID PRIMARY KEY REFERENCES clinics(id) ON DELETE CASCADE,
    config     JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
