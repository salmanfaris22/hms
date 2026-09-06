-- The Add Medication form (3820:66132) captures a purpose and a prescribing
-- doctor, and the Medications rail panel prints both — neither had a column.
-- Migrations re-run on every boot, so these must stay idempotent.
ALTER TABLE patient_medications ADD COLUMN IF NOT EXISTS purpose TEXT NOT NULL DEFAULT '';
ALTER TABLE patient_medications ADD COLUMN IF NOT EXISTS doctor  TEXT NOT NULL DEFAULT '';
