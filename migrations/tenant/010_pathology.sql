-- Clinical Catalogue: pathology tests with parameters

CREATE TABLE IF NOT EXISTS pathology_categories (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX        IF NOT EXISTS pathology_categories_clinic_idx     ON pathology_categories (clinic_id);
CREATE UNIQUE INDEX IF NOT EXISTS pathology_categories_clinic_name_ux ON pathology_categories (clinic_id, lower(name));

CREATE TABLE IF NOT EXISTS pathology_tests (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id     UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    test_name     TEXT NOT NULL,
    test_code    TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT '',
    test_type    TEXT NOT NULL DEFAULT '',
    description TEXT NOT NULL DEFAULT '',
    price       NUMERIC(12,2) NOT NULL DEFAULT 0,
    tax         TEXT NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS pathology_tests_clinic_idx      ON pathology_tests (clinic_id);
CREATE INDEX IF NOT EXISTS pathology_tests_clinic_name_idx ON pathology_tests (clinic_id, lower(test_name));
CREATE INDEX IF NOT EXISTS pathology_tests_clinic_cat_idx  ON pathology_tests (clinic_id, category);

-- Use existing column names from partial migration
CREATE TABLE IF NOT EXISTS pathology_parameters (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id       UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    parameter_name  TEXT NOT NULL,
    units          TEXT NOT NULL DEFAULT '',
    ref_min        NUMERIC(12,2) NOT NULL DEFAULT 0,
    ref_max        NUMERIC(12,2) NOT NULL DEFAULT 0,
    method         TEXT NOT NULL DEFAULT '',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS pathology_parameters_clinic_idx ON pathology_parameters (clinic_id);
CREATE INDEX IF NOT EXISTS pathology_parameters_clinic_name_idx ON pathology_parameters (clinic_id, lower(parameter_name));

CREATE TABLE IF NOT EXISTS pathology_parameter_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id   UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    group_name TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS pathology_parameter_groups_clinic_idx ON pathology_parameter_groups (clinic_id);
CREATE UNIQUE INDEX IF NOT EXISTS pathology_parameter_groups_clinic_name_ux ON pathology_parameter_groups (clinic_id, lower(group_name));

CREATE TABLE IF NOT EXISTS pathology_group_parameters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID NOT NULL REFERENCES pathology_parameter_groups(id) ON DELETE CASCADE,
    param_id   UUID NOT NULL REFERENCES pathology_parameters(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(group_id, param_id)
);
CREATE INDEX IF NOT EXISTS pathology_group_parameters_group_idx ON pathology_group_parameters (group_id);

CREATE TABLE IF NOT EXISTS pathology_test_parameters (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id    UUID NOT NULL REFERENCES pathology_tests(id) ON DELETE CASCADE,
    param_id  UUID NOT NULL REFERENCES pathology_parameters(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(test_id, param_id)
);
CREATE INDEX IF NOT EXISTS pathology_test_parameters_test_idx ON pathology_test_parameters (test_id);

CREATE TABLE IF NOT EXISTS pathology_test_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    test_id    UUID NOT NULL REFERENCES pathology_tests(id) ON DELETE CASCADE,
    group_id   UUID NOT NULL REFERENCES pathology_parameter_groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(test_id, group_id)
);
CREATE INDEX IF NOT EXISTS pathology_test_groups_test_idx ON pathology_test_groups (test_id);