-- Add default follow-up count/days columns and update set_by to support 'default' mode.
ALTER TABLE appointment_settings
    ADD COLUMN IF NOT EXISTS default_followup_count INT NOT NULL DEFAULT 2,
    ADD COLUMN IF NOT EXISTS default_followup_days  INT NOT NULL DEFAULT 30;

-- Widen the comment to reflect new valid value
COMMENT ON COLUMN appointment_settings.followup_set_by IS '''default'' | ''doctor'' | ''department''';
