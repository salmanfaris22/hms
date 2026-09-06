-- Patient profile satellites: alerts, allergies, medications, vitals,
-- appointments, treatments, invoices, insurance claims, and a polymorphic
-- timeline table that the profile page pulls into one stream.

CREATE TABLE IF NOT EXISTS patient_alerts (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    severity   TEXT NOT NULL DEFAULT 'medium', -- 'high' | 'medium' | 'low'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS patient_alerts_patient_idx ON patient_alerts (patient_id);

CREATE TABLE IF NOT EXISTS patient_allergy_entries (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    severity   TEXT NOT NULL DEFAULT 'medium',
    reaction   TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS patient_allergy_entries_patient_idx ON patient_allergy_entries (patient_id);

CREATE TABLE IF NOT EXISTS patient_medications (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    dose       TEXT NOT NULL DEFAULT '',
    schedule   TEXT NOT NULL DEFAULT '',
    started_at DATE,
    status     TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS patient_medications_patient_idx ON patient_medications (patient_id);

CREATE TABLE IF NOT EXISTS patient_vitals (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    category    TEXT NOT NULL DEFAULT '', -- 'blood' | 'heart' | 'brain' | 'lungs' | 'kidney'
    kind        TEXT NOT NULL,             -- 'blood_sugar', 'blood_pressure', ...
    value_text  TEXT NOT NULL,
    unit        TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'normal' -- 'normal' | 'high' | 'low' | 'elevated'
);
CREATE INDEX IF NOT EXISTS patient_vitals_patient_idx ON patient_vitals (patient_id, recorded_at DESC);

CREATE TABLE IF NOT EXISTS appointments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id     UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    patient_id    UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    provider_name TEXT NOT NULL DEFAULT '',
    kind          TEXT NOT NULL DEFAULT '',
    scheduled_at  TIMESTAMPTZ NOT NULL,
    duration_min  INT NOT NULL DEFAULT 30,
    status        TEXT NOT NULL DEFAULT 'scheduled', -- 'scheduled' | 'completed' | 'cancelled'
    notes         TEXT NOT NULL DEFAULT '',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS appointments_patient_idx ON appointments (patient_id, scheduled_at DESC);

CREATE TABLE IF NOT EXISTS patient_treatments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id   UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    category     TEXT NOT NULL DEFAULT '',
    amount_cents INT NOT NULL DEFAULT 0,
    performed_at DATE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS patient_treatments_patient_idx ON patient_treatments (patient_id, performed_at DESC);

CREATE TABLE IF NOT EXISTS invoices (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id           UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    patient_id          UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    number              TEXT NOT NULL DEFAULT '',
    amount_total_cents  INT NOT NULL DEFAULT 0,
    amount_paid_cents   INT NOT NULL DEFAULT 0,
    amount_due_cents    INT NOT NULL DEFAULT 0,
    status              TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'paid' | 'overdue'
    issued_at           DATE,
    due_at              DATE,
    line_items          JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS invoices_patient_idx ON invoices (patient_id, issued_at DESC);

CREATE TABLE IF NOT EXISTS insurance_claims (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id            UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    provider              TEXT NOT NULL DEFAULT '',
    policy_number         TEXT NOT NULL DEFAULT '',
    claim_number          TEXT NOT NULL DEFAULT '',
    amount_claimed_cents  INT NOT NULL DEFAULT 0,
    amount_approved_cents INT NOT NULL DEFAULT 0,
    amount_due_cents      INT NOT NULL DEFAULT 0,
    status                TEXT NOT NULL DEFAULT 'submitted',
    submitted_at          DATE,
    resolved_at           DATE,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS insurance_claims_patient_idx ON insurance_claims (patient_id);

-- Polymorphic timeline — every module appends here so the profile page can
-- merge EMR, billing, insurance, labs, etc. into a single chronological feed.
CREATE TABLE IF NOT EXISTS patient_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id   UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    kind         TEXT NOT NULL, -- 'visit' | 'emr' | 'prescription' | 'lab' | 'invoice' | 'payment' | 'insurance_claim' | 'note'
    title        TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    amount_cents INT,
    metadata     JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS patient_events_patient_idx ON patient_events (patient_id, occurred_at DESC);
