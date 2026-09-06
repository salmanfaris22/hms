-- Per-clinic auto-numbering configuration for patient IDs, invoices, bills, etc.
CREATE TABLE IF NOT EXISTS numbering_settings (
    clinic_id  UUID PRIMARY KEY REFERENCES clinics(id) ON DELETE CASCADE,
    settings   JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
