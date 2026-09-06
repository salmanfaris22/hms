-- A draft bill is reopened and carried on with (Generate Bill, 4723:68314).
-- Only the rounding *result* was stored, so a draft came back with its rounding
-- rule forgotten and its total quietly changed on the next save. Keep the rule.
ALTER TABLE invoices ADD COLUMN IF NOT EXISTS round_off_mode TEXT NOT NULL DEFAULT 'none';
