-- The appointment calendar (3424:46141) needs three things the table never had:
-- a column per *doctor* rather than a free-text provider name, an EMRG flag, and
-- somewhere to record the blocked slots the "Blocked" cells draw.
--
-- provider_name stays: appointments booked before this, and any booked against
-- someone who is not a staff user, still have to render.
--
-- Every migration re-runs on boot, so all of this must stay idempotent.
ALTER TABLE appointments
  ADD COLUMN IF NOT EXISTS provider_id  UUID REFERENCES users(id) ON DELETE SET NULL,
  ADD COLUMN IF NOT EXISTS is_emergency BOOLEAN NOT NULL DEFAULT false;

-- the calendar reads one clinic's day at a time
CREATE INDEX IF NOT EXISTS appointments_clinic_day_idx
    ON appointments (clinic_id, scheduled_at);

-- A block covers a span of the calendar. provider_id NULL blocks every column,
-- which is what "Block Calendar" does for a clinic-wide closure.
CREATE TABLE IF NOT EXISTS appointment_blocks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id   UUID        NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    provider_id UUID        REFERENCES users(id) ON DELETE CASCADE,
    starts_at   TIMESTAMPTZ NOT NULL,
    ends_at     TIMESTAMPTZ NOT NULL,
    reason      TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT appointment_blocks_span CHECK (ends_at > starts_at)
);

CREATE INDEX IF NOT EXISTS appointment_blocks_clinic_day_idx
    ON appointment_blocks (clinic_id, starts_at);
