-- The "Event Slot" tab of the booking form (4691:62738) reserves a doctor's time
-- with no patient at all — it has no patient field. Every other tab still
-- requires one; the service enforces that, since the column can no longer.
--
-- Every migration re-runs on boot, and DROP NOT NULL on an already-nullable
-- column is a no-op, so this is idempotent.
ALTER TABLE appointments ALTER COLUMN patient_id DROP NOT NULL;
