-- Optional phone number on tenant users.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS phone         TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS phone_country TEXT NOT NULL DEFAULT '';
