-- The queue (3453:44305) tracks an appointment through the day: it is checked
-- in, the visit starts, then it is checked out. It shows how long someone has
-- been waiting and how long the session has run, so those moments need stamping
-- rather than inferring, and each checked-in patient gets a token.
--
-- Every migration re-runs on boot, so this must stay idempotent.
ALTER TABLE appointments
  ADD COLUMN IF NOT EXISTS checked_in_at    TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS visit_started_at TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS visit_ended_at   TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS token            TEXT NOT NULL DEFAULT '';
