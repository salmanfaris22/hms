-- A report is a finished result (3765:52465). The Reports tab lists completed
-- tests by their own number and by whether anything came back out of band, so
-- both have to be stored rather than recomputed from the values every time a
-- list is drawn.
--
-- Every migration re-runs on boot, so this must stay idempotent.
ALTER TABLE lab_order_tests ADD COLUMN IF NOT EXISTS report_number TEXT NOT NULL DEFAULT '';
-- '' until reported, then 'normal' | 'abnormal'
ALTER TABLE lab_order_tests ADD COLUMN IF NOT EXISTS result_flag TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS lab_order_tests_report_idx
    ON lab_order_tests (clinic_id, reported_at DESC)
    WHERE status = 'completed';
