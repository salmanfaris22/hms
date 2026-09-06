-- The prescription pad (3467:53677) is one document per appointment: vitals,
-- complaints, diagnosis, the medicines, advice, the dental findings and the
-- follow-up. The sections are a long list of free-form entries that the design
-- keeps adding to, so the document is stored as JSONB rather than a column per
-- field — the shape belongs to the pad, not to the schema.
--
-- Every migration re-runs on boot, so this must stay idempotent.
CREATE TABLE IF NOT EXISTS patient_prescriptions (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id      UUID        NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    patient_id     UUID        NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    appointment_id UUID        REFERENCES appointments(id) ON DELETE SET NULL,
    doc            JSONB       NOT NULL DEFAULT '{}'::jsonb,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- one pad per appointment, so reopening it edits rather than duplicates
CREATE UNIQUE INDEX IF NOT EXISTS patient_prescriptions_appointment_idx
    ON patient_prescriptions (appointment_id) WHERE appointment_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS patient_prescriptions_patient_idx
    ON patient_prescriptions (patient_id, created_at DESC);
