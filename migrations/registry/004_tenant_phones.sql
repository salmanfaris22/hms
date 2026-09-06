-- Multi-phone contact list for a hospital organisation.
-- Kept in addition to the single-phone columns for backward compatibility.

ALTER TABLE tenants
    ADD COLUMN IF NOT EXISTS phones JSONB NOT NULL DEFAULT '[]'::jsonb;
