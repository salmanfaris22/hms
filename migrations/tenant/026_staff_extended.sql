-- Extend staff_profiles with personal, address, and employment fields
ALTER TABLE staff_profiles
  ADD COLUMN IF NOT EXISTS father_name       TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS mother_name       TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS gender            TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS marital_status    TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS date_of_birth     DATE,
  ADD COLUMN IF NOT EXISTS blood_group       TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS professional_id   TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS address           TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS locality          TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS pincode           TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS state             TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS country           TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS date_of_leaving   DATE;

-- Weekly schedule timetable for staff members
CREATE TABLE IF NOT EXISTS staff_weekly_schedule (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    day_of_week  SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6), -- 0=Sunday … 6=Saturday
    start_time   TIME NOT NULL,
    end_time     TIME NOT NULL,
    is_active    BOOLEAN NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (user_id, day_of_week, start_time)
);

CREATE INDEX IF NOT EXISTS idx_staff_schedule_user ON staff_weekly_schedule(user_id);
CREATE INDEX IF NOT EXISTS idx_staff_schedule_day  ON staff_weekly_schedule(day_of_week);

-- Staff weekly schedule API endpoints
-- GET    /staff/:id/schedule     → list schedule slots for a staff member
-- PUT    /staff/:id/schedule     → replace all schedule slots for a staff member
