-- Clinical Catalogue: radiology tests with per-clinic category lookup.

CREATE TABLE IF NOT EXISTS radiology_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        IF NOT EXISTS radiology_categories_clinic_idx     ON radiology_categories (clinic_id);
CREATE UNIQUE INDEX IF NOT EXISTS radiology_categories_clinic_name_ux ON radiology_categories (clinic_id, lower(name));

CREATE TABLE IF NOT EXISTS radiology_tests (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id   UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    test_name   TEXT NOT NULL,
    test_code   TEXT NOT NULL DEFAULT '',
    category    TEXT NOT NULL DEFAULT '',
    body_part   TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    price       NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax         TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS radiology_tests_clinic_idx      ON radiology_tests (clinic_id);
CREATE INDEX IF NOT EXISTS radiology_tests_clinic_name_idx ON radiology_tests (clinic_id, lower(test_name));
CREATE INDEX IF NOT EXISTS radiology_tests_clinic_cat_idx  ON radiology_tests (clinic_id, category);
