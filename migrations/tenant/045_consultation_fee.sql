-- The fee a clinic collects at the time of booking, and what was taken for it.
--
-- The booking dialog offers "Collect Consultation Fee" for every booking mode
-- except an Event Slot, which reserves time without a patient and so has nobody
-- to charge. The amount is a clinic-wide default; the split records how it was
-- actually paid, which may be across more than one method.

ALTER TABLE appointment_settings
  ADD COLUMN IF NOT EXISTS consultation_fee_cents integer NOT NULL DEFAULT 0;

-- What was collected against one appointment. The invoice carries the money;
-- this ties it back to the booking it was taken at.
ALTER TABLE invoices
  ADD COLUMN IF NOT EXISTS appointment_id uuid REFERENCES appointments(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS invoices_appointment_idx ON invoices (appointment_id);
