-- Links a patient to their category and special status.
--
-- 020_patient_categories.sql created both catalogues with a patient_count
-- column, but nothing on `patients` ever pointed at them, so the Patient List's
-- PATIENT GROUP column and its VIP badge had no data source at all.
ALTER TABLE patients
  ADD COLUMN IF NOT EXISTS category_id       UUID REFERENCES patient_categories(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS special_status_id UUID REFERENCES special_statuses(id)   ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS patients_category_idx ON patients (clinic_id, category_id);
