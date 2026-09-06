-- Optimized indexes for 10K+ users

-- Composite index for patient list queries (covers most common queries)
CREATE INDEX IF NOT EXISTS patients_list_idx ON patients (clinic_id, status, created_at DESC);

-- Covering index for patient get by number
CREATE INDEX IF NOT EXISTS patients_number_idx ON patients (clinic_id, patient_number);

-- Index for search (used in LIKE queries)
CREATE INDEX IF NOT EXISTS patients_search_idx ON patients (clinic_id, lower(first_name), lower(last_name));

-- Clinic members composite lookup
CREATE INDEX IF NOT EXISTS clinic_members_lookup_idx ON clinic_members (user_id, clinic_id);

-- Appointments indexes
CREATE INDEX IF NOT EXISTS appointments_patient_idx ON appointments (patient_id, scheduled_at DESC);
CREATE INDEX IF NOT EXISTS appointments_clinic_idx ON appointments (clinic_id, status, scheduled_at DESC);

-- Invoices indexes
CREATE INDEX IF NOT EXISTS invoices_patient_idx ON invoices (patient_id, status);
CREATE INDEX IF NOT EXISTS invoices_clinic_idx ON invoices (clinic_id, status, issued_at DESC);

-- Patient vitals indexes
CREATE INDEX IF NOT EXISTS patient_vitals_patient_idx ON patient_vitals (patient_id, recorded_at DESC);
CREATE INDEX IF NOT EXISTS patient_vitals_category_idx ON patient_vitals (patient_id, category, recorded_at DESC);

-- Enable pg_stat_statements for query analysis
CREATE EXTENSION IF NOT EXISTS pg_stat_statements;

-- Analyze tables for query planner
ANALYZE patients;
ANALYZE clinics;
ANALYZE users;
ANALYZE clinic_members;
ANALYZE appointments;
ANALYZE invoices;