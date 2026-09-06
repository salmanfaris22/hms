-- Money handed back (4565:73203). A refund is not a negative payment: it is its
-- own record, with its own reason, so the till and the patient's history both
-- show what actually happened rather than a payment that quietly shrank.
CREATE TABLE IF NOT EXISTS invoice_refunds (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id    UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    invoice_id   UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    patient_id   UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    method       TEXT NOT NULL DEFAULT '',
    note         TEXT NOT NULL DEFAULT '',
    refunded_by  UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS invoice_refunds_invoice_idx ON invoice_refunds (invoice_id, created_at DESC);
CREATE INDEX IF NOT EXISTS invoice_refunds_patient_idx ON invoice_refunds (patient_id, created_at DESC);
