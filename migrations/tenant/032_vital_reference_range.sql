-- The Health Overview card (2857:52847) prints a "Normal Range" pill beside the
-- headline vital. Nothing on patient_vitals carried one, so the pill had no
-- data to render from. Blank means "no range on file" and the pill is omitted.
ALTER TABLE patient_vitals ADD COLUMN IF NOT EXISTS reference_range TEXT NOT NULL DEFAULT '';
