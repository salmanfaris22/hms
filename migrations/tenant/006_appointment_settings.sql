-- Per-clinic appointment settings. Each row is a single "preferences" blob
-- populated on demand; upsert-style writes from the settings screen.

CREATE TABLE IF NOT EXISTS appointment_settings (
    clinic_id                   UUID PRIMARY KEY REFERENCES clinics(id) ON DELETE CASCADE,
    slot_duration_min           INT NOT NULL DEFAULT 30,
    online_booking_confirmation BOOLEAN NOT NULL DEFAULT false,
    free_followup_enabled       BOOLEAN NOT NULL DEFAULT false,
    followup_set_by             TEXT NOT NULL DEFAULT 'department', -- 'department' | 'doctor'
    -- [{ "name": "Cardiology", "count": 3, "days": 3 }, ...]
    followup_rules              JSONB NOT NULL DEFAULT '[]'::jsonb,
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);
