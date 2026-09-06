-- The Laboratory module (3765:51814). The catalogue already answers "what tests
-- can this clinic run"; this answers "who was tested, with what sample, and what
-- came back".
--
-- An order is what a clinician asks for in one go; a row of lab_order_tests is
-- one test on that order, and that is what the worklist lists, what a sample is
-- collected against, and what a result is entered into. Each test carries its
-- own status because a blood draw and an X-ray on one order do not progress
-- together.
--
-- Every migration re-runs on boot, so this must stay idempotent.

CREATE TABLE IF NOT EXISTS lab_orders (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id     UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    patient_id    UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    order_number  TEXT NOT NULL DEFAULT '',
    -- 'routine' | 'urgent' | 'stat' — the frame's three pills
    priority      TEXT NOT NULL DEFAULT 'routine',
    -- who asked for it: a staff member where one was picked, else a typed name
    ordered_by    TEXT NOT NULL DEFAULT '',
    ordered_by_id UUID,
    -- the patient as they were at the time: reference bands are read by age,
    -- sex and patient category, and a birthday later must not restate an old
    -- result as abnormal
    patient_age      INT,
    patient_sex      TEXT NOT NULL DEFAULT '',
    patient_category TEXT NOT NULL DEFAULT '',
    -- the visit it was raised from, where it was
    appointment_id UUID,
    notes         TEXT NOT NULL DEFAULT '',
    ordered_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS lab_order_tests (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id    UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    order_id     UUID NOT NULL REFERENCES lab_orders(id) ON DELETE CASCADE,
    -- 'pathology' | 'radiology': the two catalogues a test can come from
    source       TEXT NOT NULL DEFAULT 'pathology',
    -- the catalogue row it was chosen from; kept nullable because a test may be
    -- retired from the catalogue long after it was run
    test_id      UUID,
    -- copied at order time, so an old order still reads as it was ordered
    test_name    TEXT NOT NULL DEFAULT '',
    test_code    TEXT NOT NULL DEFAULT '',
    category     TEXT NOT NULL DEFAULT '',
    -- the specimen for pathology, the body part for radiology
    sample_type  TEXT NOT NULL DEFAULT '',
    price_cents  INT NOT NULL DEFAULT 0,
    tax_pct      NUMERIC(6,2) NOT NULL DEFAULT 0,
    -- 'pending' (no sample) | 'in_progress' (sample taken) | 'completed'
    -- (result entered) | 'cancelled'
    status       TEXT NOT NULL DEFAULT 'pending',
    collected_at TIMESTAMPTZ,
    collected_by TEXT NOT NULL DEFAULT '',
    -- one row per parameter, as entered: name, value, unit, band and flag
    result       JSONB NOT NULL DEFAULT '[]'::jsonb,
    result_notes TEXT NOT NULL DEFAULT '',
    reported_at  TIMESTAMPTZ,
    reported_by  TEXT NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS lab_orders_clinic_idx      ON lab_orders (clinic_id, ordered_at DESC);
CREATE INDEX IF NOT EXISTS lab_orders_patient_idx     ON lab_orders (patient_id, ordered_at DESC);
CREATE INDEX IF NOT EXISTS lab_order_tests_order_idx  ON lab_order_tests (order_id);
CREATE INDEX IF NOT EXISTS lab_order_tests_clinic_idx ON lab_order_tests (clinic_id, status, created_at DESC);
