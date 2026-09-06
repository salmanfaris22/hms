-- The Documents tab (3151:46605) lists a patient's files with a category, the
-- clinician they came from and the date on the document itself, which can differ
-- from when it was uploaded. Only staff_documents existed before this.
-- Migrations re-run on every boot, so this must stay idempotent.
CREATE TABLE IF NOT EXISTS patient_documents (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id    UUID        NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    url           TEXT        NOT NULL,
    mime_type     TEXT        NOT NULL DEFAULT '',
    size_bytes    BIGINT      NOT NULL DEFAULT 0,
    category      TEXT        NOT NULL DEFAULT 'Other',
    doctor        TEXT        NOT NULL DEFAULT '',
    document_date DATE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS patient_documents_patient_idx
    ON patient_documents (patient_id, created_at DESC);
