-- Clinical Catalogue: drugs & consumables with per-clinic lookup tables
-- for categories, manufacturers and units.

CREATE TABLE IF NOT EXISTS drug_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        IF NOT EXISTS drug_categories_clinic_idx     ON drug_categories (clinic_id);
CREATE UNIQUE INDEX IF NOT EXISTS drug_categories_clinic_name_ux ON drug_categories (clinic_id, lower(name));

CREATE TABLE IF NOT EXISTS drug_manufacturers (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        IF NOT EXISTS drug_manufacturers_clinic_idx     ON drug_manufacturers (clinic_id);
CREATE UNIQUE INDEX IF NOT EXISTS drug_manufacturers_clinic_name_ux ON drug_manufacturers (clinic_id, lower(name));

CREATE TABLE IF NOT EXISTS drug_units (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    kind       TEXT NOT NULL, -- 'primary' | 'secondary'
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        IF NOT EXISTS drug_units_clinic_idx     ON drug_units (clinic_id, kind);
CREATE UNIQUE INDEX IF NOT EXISTS drug_units_clinic_name_ux ON drug_units (clinic_id, kind, lower(name));

CREATE TABLE IF NOT EXISTS drugs (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id      UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    drug_name      TEXT NOT NULL,
    generic_name   TEXT NOT NULL DEFAULT '',
    category       TEXT NOT NULL DEFAULT '',
    strength       TEXT NOT NULL DEFAULT '',
    item_code      TEXT NOT NULL DEFAULT '',
    manufacturer   TEXT NOT NULL DEFAULT '',
    instruction    TEXT NOT NULL DEFAULT '',
    primary_unit   TEXT NOT NULL DEFAULT '',
    secondary_unit TEXT NOT NULL DEFAULT '',
    reorder_level  TEXT NOT NULL DEFAULT '',
    hsn_code       TEXT NOT NULL DEFAULT '',
    tax            TEXT NOT NULL DEFAULT '',
    discount       TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS drugs_clinic_idx      ON drugs (clinic_id);
CREATE INDEX IF NOT EXISTS drugs_clinic_name_idx ON drugs (clinic_id, lower(drug_name));
CREATE INDEX IF NOT EXISTS drugs_clinic_cat_idx  ON drugs (clinic_id, category);
