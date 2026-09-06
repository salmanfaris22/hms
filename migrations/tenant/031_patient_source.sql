-- 2719:36180 adds two fields to the Add Patient form that had no column:
-- "Source" (how the patient reached the clinic) and "Insurance Validity".
-- Every migration re-runs on boot, so both statements must be idempotent.
ALTER TABLE patients ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT '';
ALTER TABLE patients ADD COLUMN IF NOT EXISTS insurance_validity DATE;
