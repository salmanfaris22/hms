-- "Create New Category" (1336:20803) lists the clinic's custom document categories
-- and deletes them one by one, so they have to outlive the dialog that made them.
-- The built-in six (Imaging, Lab Report, …) stay in the frontend; only the
-- clinic's own additions live here.
--
-- Every migration re-runs on boot, so both statements must be idempotent.
CREATE TABLE IF NOT EXISTS patient_document_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID        NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One name per clinic, case-insensitively: "X-Ray" and "x-ray" are the same category.
CREATE UNIQUE INDEX IF NOT EXISTS patient_document_categories_clinic_name_idx
    ON patient_document_categories (clinic_id, lower(name));
