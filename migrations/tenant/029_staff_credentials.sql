-- Records when a staff member's login was explicitly set.
--
-- CreateStaff always writes a password_hash (defaulting to a shared temporary
-- password), so the hash alone cannot tell an established account from one that
-- has never had credentials chosen for it. User Info & Security needs that
-- distinction to decide between its "Set Credential" and "Change Password" forms.
ALTER TABLE staff_profiles
  ADD COLUMN IF NOT EXISTS credentials_set_at TIMESTAMPTZ;

-- Accounts that predate this column have no stamp, so they would all read as
-- "no credentials". Anyone who has actually logged in demonstrably has a working
-- login, so seed them from that. Idempotent: this runner re-executes every
-- migration on boot, and the WHERE clause skips rows already stamped.
UPDATE staff_profiles sp
   SET credentials_set_at = u.last_login_at
  FROM users u
 WHERE u.id = sp.user_id
   AND sp.credentials_set_at IS NULL
   AND u.last_login_at IS NOT NULL;
