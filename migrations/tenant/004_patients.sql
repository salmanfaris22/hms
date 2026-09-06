-- Patient records + per-clinic form field configuration.

CREATE TABLE IF NOT EXISTS patients (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id           UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    patient_number      TEXT NOT NULL DEFAULT '',

    -- Personal
    first_name          TEXT NOT NULL,
    middle_name         TEXT NOT NULL DEFAULT '',
    last_name           TEXT NOT NULL DEFAULT '',
    date_of_birth       DATE,
    sex                 TEXT NOT NULL DEFAULT '',
    blood_type          TEXT NOT NULL DEFAULT '',
    marital_status      TEXT NOT NULL DEFAULT '',
    photo_url           TEXT NOT NULL DEFAULT '',

    -- Contact
    phones              JSONB NOT NULL DEFAULT '[]'::jsonb,
    email               TEXT NOT NULL DEFAULT '',
    address_line        TEXT NOT NULL DEFAULT '',
    city                TEXT NOT NULL DEFAULT '',
    state               TEXT NOT NULL DEFAULT '',
    country             TEXT NOT NULL DEFAULT '',
    zip_code            TEXT NOT NULL DEFAULT '',

    -- Emergency contact
    emergency_name      TEXT NOT NULL DEFAULT '',
    emergency_phone     TEXT NOT NULL DEFAULT '',

    -- Medical
    allergies           TEXT NOT NULL DEFAULT '',
    medical_conditions  TEXT NOT NULL DEFAULT '',

    -- Insurance
    insurance_provider  TEXT NOT NULL DEFAULT '',
    insurance_id        TEXT NOT NULL DEFAULT '',

    -- Derived / display
    condition           TEXT NOT NULL DEFAULT '',
    last_visit_at       DATE,
    status              TEXT NOT NULL DEFAULT 'active', -- 'active' | 'inactive'
    notes               TEXT NOT NULL DEFAULT '',
    family_members      JSONB NOT NULL DEFAULT '[]'::jsonb,

    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS patients_clinic_idx ON patients (clinic_id);
CREATE INDEX IF NOT EXISTS patients_status_idx ON patients (clinic_id, status);
CREATE INDEX IF NOT EXISTS patients_name_idx   ON patients (clinic_id, lower(first_name), lower(last_name));

-- Auto-number: PA000001, PA000002, ... per clinic
CREATE SEQUENCE IF NOT EXISTS patient_number_seq;

-- Per-clinic form field configuration. Stored as JSONB: { "<fieldKey>": { "visible": bool, "required": bool } }
CREATE TABLE IF NOT EXISTS patient_field_config (
    clinic_id  UUID PRIMARY KEY REFERENCES clinics(id) ON DELETE CASCADE,
    config     JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
