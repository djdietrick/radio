-- Add authentication fields to users. The seeded default user (from 0001) is
-- bootstrapped into an admin account at startup from env credentials, keeping
-- ownership of any pre-auth playlists/stations.

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS password_hash TEXT NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS is_admin      BOOLEAN NOT NULL DEFAULT FALSE;

-- username is the login identifier; enforce it is present and unique. (0001
-- already made it UNIQUE; ensure it cannot be null going forward.)
ALTER TABLE users
    ALTER COLUMN username SET NOT NULL;
