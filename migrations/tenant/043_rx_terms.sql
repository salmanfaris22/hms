-- Terms the prescription pad offers as suggestions: chief complaints today,
-- and the pad's other free-text sections as they are built out. "Add New" in
-- the pad writes here, so a clinic's own vocabulary grows with use.

CREATE TABLE IF NOT EXISTS rx_terms (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        IF NOT EXISTS rx_terms_clinic_kind_idx  ON rx_terms (clinic_id, kind);
CREATE UNIQUE INDEX IF NOT EXISTS rx_terms_clinic_name_ux   ON rx_terms (clinic_id, kind, lower(name));
