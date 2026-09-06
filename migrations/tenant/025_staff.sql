-- Staff Management: extend users with profile fields, plus roles + permissions.

CREATE TABLE IF NOT EXISTS staff_profiles (
    user_id              UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    staff_code           TEXT NOT NULL,
    profile_photo        TEXT NOT NULL DEFAULT '',
    first_name           TEXT NOT NULL DEFAULT '',
    middle_name          TEXT NOT NULL DEFAULT '',
    last_name            TEXT NOT NULL DEFAULT '',
    department_id        UUID REFERENCES departments(id) ON DELETE SET NULL,
    designation_id       UUID REFERENCES designations(id) ON DELETE SET NULL,
    specialization_id    UUID REFERENCES specializations(id) ON DELETE SET NULL,
    mobile_country       TEXT NOT NULL DEFAULT '+91',
    mobile_number        TEXT NOT NULL DEFAULT '',
    additional_mobile    TEXT NOT NULL DEFAULT '',
    landline_number      TEXT NOT NULL DEFAULT '',
    view_in_emr          BOOLEAN NOT NULL DEFAULT true,
    join_date            DATE NOT NULL DEFAULT CURRENT_DATE,
    status               TEXT NOT NULL DEFAULT 'active',  -- active | inactive | on_leave
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (staff_code)
);

CREATE TABLE IF NOT EXISTS staff_clinics (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, clinic_id)
);
CREATE INDEX IF NOT EXISTS staff_clinics_clinic_idx ON staff_clinics (clinic_id);

CREATE TABLE IF NOT EXISTS staff_documents (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    url         TEXT NOT NULL,
    mime_type   TEXT NOT NULL DEFAULT '',
    size_bytes  BIGINT NOT NULL DEFAULT 0,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS staff_documents_user_idx ON staff_documents (user_id);

CREATE TABLE IF NOT EXISTS roles (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id    UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    name         TEXT NOT NULL,
    description  TEXT NOT NULL DEFAULT '',
    permissions  JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (clinic_id, name)
);

CREATE TABLE IF NOT EXISTS user_roles (
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id    UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    clinic_id  UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    PRIMARY KEY (user_id, role_id, clinic_id)
);
CREATE INDEX IF NOT EXISTS user_roles_role_idx ON user_roles (role_id);
CREATE INDEX IF NOT EXISTS user_roles_clinic_idx ON user_roles (clinic_id);
