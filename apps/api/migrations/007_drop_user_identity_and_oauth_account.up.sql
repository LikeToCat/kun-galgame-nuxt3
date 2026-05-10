-- 007: After OAuth becomes the single identity source, finish the kungal-side
--      cleanup:
--
--   1. Stand up `kungal_user_state` (the slim local table holding moemoepoint
--      and daily counters — see internal/user/model/state.go).
--   2. Carry the existing data from "user".{moemoepoint, daily_*} into the new
--      table. user.id is already aligned with OAuth.users.id by the
--      migrate-users script (id-unification step 7), so no remap is needed.
--   3. Drop the obsolete oauth_account middle table — /oauth/userinfo now
--      returns an integer id aligned with the local user.id, so the
--      sub→id indirection table is dead weight (per
--      docs/migration/user/08-downstream-integration.md §7.1).
--   4. Drop OAuth-owned identity columns from "user" (name / email / password
--      / avatar / bio / role / status / ip).
--   5. Drop moemoepoint / daily_* from "user" (now living in
--      kungal_user_state).
--
-- Run AFTER:
--   - The OAuth-side migrate-users script has aligned three-database IDs and
--     populated OAuth.users with the authoritative identity records.
--   - Migrations 005 (wiki-managed cleanup) and 006 (resource provider name)
--     plus the backfill-provider-names tool have run, so the kungal schema
--     is otherwise stable.
--
-- WARNING: Not reversible (data in dropped columns is preserved in
--          kungal_user_state for moemoepoint/daily_* but lost for identity
--          fields — those live in OAuth now).

BEGIN;

-- ──────────────────────────────────────────
-- 1. Create kungal_user_state if missing.
-- ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS kungal_user_state (
    user_id                    INT PRIMARY KEY,
    moemoepoint                INT NOT NULL DEFAULT 7,
    daily_check_in             INT NOT NULL DEFAULT 0,
    daily_image_count          INT NOT NULL DEFAULT 0,
    daily_toolset_upload_count INT NOT NULL DEFAULT 0,
    created                    TIMESTAMP NOT NULL DEFAULT NOW(),
    updated                    TIMESTAMP NOT NULL DEFAULT NOW()
);

-- ──────────────────────────────────────────
-- 2. Carry per-site state from "user" → kungal_user_state.
--    Idempotent: re-running this migration after a partial failure does not
--    overwrite already-carried rows (ON CONFLICT DO NOTHING).
--    Note: this references columns that step 4 will drop. The COMMIT at the
--    end is what makes the carry-over visible before the drop happens within
--    the same transaction; since both run in one tx, we need to do the
--    SELECT before the ALTER. PostgreSQL is fine with this ordering.
-- ──────────────────────────────────────────
INSERT INTO kungal_user_state (
    user_id, moemoepoint,
    daily_check_in, daily_image_count, daily_toolset_upload_count,
    created, updated
)
SELECT
    id,
    COALESCE(moemoepoint, 7),
    COALESCE(daily_check_in, 0),
    COALESCE(daily_image_count, 0),
    COALESCE(daily_toolset_upload_count, 0),
    COALESCE(created, NOW()),
    COALESCE(updated, NOW())
FROM "user"
ON CONFLICT (user_id) DO NOTHING;

-- ──────────────────────────────────────────
-- 3. Drop the OAuth-link middle table.
-- ──────────────────────────────────────────
DROP TABLE IF EXISTS oauth_account;

-- ──────────────────────────────────────────
-- 4. Drop OAuth-owned identity columns from "user".
--    Remaining: id (PK) + created/updated timestamps.
-- ──────────────────────────────────────────
ALTER TABLE "user"
  DROP COLUMN IF EXISTS name,
  DROP COLUMN IF EXISTS email,
  DROP COLUMN IF EXISTS password,
  DROP COLUMN IF EXISTS avatar,
  DROP COLUMN IF EXISTS bio,
  DROP COLUMN IF EXISTS role,
  DROP COLUMN IF EXISTS status,
  DROP COLUMN IF EXISTS ip;

-- ──────────────────────────────────────────
-- 5. Drop per-site state columns now living in kungal_user_state.
-- ──────────────────────────────────────────
ALTER TABLE "user"
  DROP COLUMN IF EXISTS moemoepoint,
  DROP COLUMN IF EXISTS daily_check_in,
  DROP COLUMN IF EXISTS daily_image_count,
  DROP COLUMN IF EXISTS daily_toolset_upload_count;

COMMIT;
