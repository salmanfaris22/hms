-- Per-clinic print settings stored as JSONB blob
CREATE TABLE IF NOT EXISTS print_settings (
    clinic_id  UUID PRIMARY KEY REFERENCES clinics(id) ON DELETE CASCADE,
    settings   JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
