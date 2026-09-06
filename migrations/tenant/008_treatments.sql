-- Clinical Catalogue: treatments & services.
-- Consumables attached to a treatment are stored inline as JSONB:
--   [{"drugId": "<uuid>", "name": "…", "quantity": 1, "dispenseMethod": "LFD"}]

CREATE TABLE IF NOT EXISTS treatments (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id      UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    treatment_name TEXT NOT NULL,
    treatment_code TEXT NOT NULL DEFAULT '',
    description    TEXT NOT NULL DEFAULT '',
    price          NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount       NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax            TEXT NOT NULL DEFAULT '',
    consumables    JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS treatments_clinic_idx      ON treatments (clinic_id);
CREATE INDEX IF NOT EXISTS treatments_clinic_name_idx ON treatments (clinic_id, lower(treatment_name));
