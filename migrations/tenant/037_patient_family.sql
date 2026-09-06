-- "Add Family Member" (3841:57713) links existing patients to each other so a
-- family shares records and billing. The link is symmetric in meaning but stored
-- once per direction, since the relation word differs each way (a patient's
-- Spouse is also their spouse, but a patient's Child is their parent's Parent).
--
-- Every migration re-runs on boot, so this must stay idempotent.
CREATE TABLE IF NOT EXISTS patient_family_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id  UUID        NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    relative_id UUID        NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    relation    TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT patient_family_not_self CHECK (patient_id <> relative_id)
);

-- one link per pair; re-adding a relative updates the relation instead
CREATE UNIQUE INDEX IF NOT EXISTS patient_family_members_pair_idx
    ON patient_family_members (patient_id, relative_id);
