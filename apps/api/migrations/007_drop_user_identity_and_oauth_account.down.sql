-- 007 down: Best-effort schema rollback. Identity-column data is gone (OAuth
-- now owns it), so this only restores the column shapes for emergency
-- compatibility, not the original values.
--
-- moemoepoint / daily_* are restored from kungal_user_state where possible.

BEGIN;

-- 1. Re-add the columns dropped in step 5 (per-site state).
ALTER TABLE "user"
  ADD COLUMN IF NOT EXISTS moemoepoint INT NOT NULL DEFAULT 7,
  ADD COLUMN IF NOT EXISTS daily_check_in INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS daily_image_count INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS daily_toolset_upload_count INT NOT NULL DEFAULT 0;

-- 2. Carry data back from kungal_user_state.
UPDATE "user" u SET
    moemoepoint                = s.moemoepoint,
    daily_check_in             = s.daily_check_in,
    daily_image_count          = s.daily_image_count,
    daily_toolset_upload_count = s.daily_toolset_upload_count
FROM kungal_user_state s
WHERE s.user_id = u.id;

-- 3. Re-add identity columns (data not recoverable; OAuth holds it now).
ALTER TABLE "user"
  ADD COLUMN IF NOT EXISTS name VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS email VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS password VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS avatar VARCHAR(255) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS bio VARCHAR(107) NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS role INT NOT NULL DEFAULT 1,
  ADD COLUMN IF NOT EXISTS status INT NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS ip VARCHAR(255) NOT NULL DEFAULT '';

-- 4. Recreate oauth_account.
CREATE TABLE IF NOT EXISTS oauth_account (
    id         SERIAL PRIMARY KEY,
    user_id    INT NOT NULL REFERENCES "user"(id) ON DELETE CASCADE,
    provider   VARCHAR(50) NOT NULL DEFAULT 'kun-oauth',
    sub        VARCHAR(255) NOT NULL,
    created    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated    TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (sub)
);
CREATE INDEX IF NOT EXISTS idx_oauth_account_user_id ON oauth_account(user_id);

-- 5. Drop kungal_user_state.
DROP TABLE IF EXISTS kungal_user_state;

COMMIT;
