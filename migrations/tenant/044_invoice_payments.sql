-- Money collected against an invoice, from the appointment's Payment & Collect
-- panel. The invoice keeps the running totals; this is the ledger behind them.

CREATE TABLE IF NOT EXISTS invoice_payments (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clinic_id    UUID NOT NULL REFERENCES clinics(id) ON DELETE CASCADE,
    invoice_id   UUID NOT NULL REFERENCES invoices(id) ON DELETE CASCADE,
    patient_id   UUID NOT NULL REFERENCES patients(id) ON DELETE CASCADE,
    amount_cents INTEGER NOT NULL CHECK (amount_cents > 0),
    method       TEXT NOT NULL DEFAULT '',
    collected_by UUID,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS invoice_payments_invoice_idx ON invoice_payments (invoice_id, created_at DESC);
CREATE INDEX IF NOT EXISTS invoice_payments_patient_idx ON invoice_payments (patient_id, created_at DESC);
