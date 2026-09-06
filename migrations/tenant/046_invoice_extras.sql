-- The invoice card (3889:55599) states Subtotal, Total Discount, Taxable Amount
-- and Total Tax and expects them to add up to the Grand Total. The extras the
-- form applies on top of the lines were only ever folded into the total, so a
-- stored invoice could not be read back line by line. Keep them.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS extra_discount_cents INT NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS extra_tax_cents      INT NOT NULL DEFAULT 0;
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS round_off_cents      INT NOT NULL DEFAULT 0;
