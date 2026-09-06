-- "Appointment per slot form" (4691:64658) records two things the table has no
-- room for: which channel the booking came through (its Offline / Online /
-- Temporary / Event Slot tabs) and who took it (its "Marked by" footer).
--
-- Every migration re-runs on boot, so this must stay idempotent.
ALTER TABLE appointments
  ADD COLUMN IF NOT EXISTS booking_mode TEXT NOT NULL DEFAULT 'offline',
  ADD COLUMN IF NOT EXISTS marked_by    TEXT NOT NULL DEFAULT '';
