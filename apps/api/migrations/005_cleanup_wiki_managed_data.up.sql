-- 005: Remove wiki-managed tables and metadata columns from galgame table.
-- After this migration, the local galgame table only stores interaction data
-- (counts, view, timestamps). All metadata is served by the wiki service.
--
-- WARNING: This migration is NOT reversible. Ensure the wiki service has
--          a complete copy of all galgame metadata before running.

BEGIN;

-- ──────────────────────────────────────────
-- 1. Drop wiki-managed tables (CASCADE for FK references)
-- ──────────────────────────────────────────

DROP TABLE IF EXISTS galgame_tag_relation CASCADE;
DROP TABLE IF EXISTS galgame_tag_alias CASCADE;
DROP TABLE IF EXISTS galgame_tag CASCADE;

DROP TABLE IF EXISTS galgame_official_relation CASCADE;
DROP TABLE IF EXISTS galgame_official_alias CASCADE;
DROP TABLE IF EXISTS galgame_official CASCADE;

DROP TABLE IF EXISTS galgame_engine_relation CASCADE;
DROP TABLE IF EXISTS galgame_engine CASCADE;

DROP TABLE IF EXISTS galgame_alias CASCADE;
DROP TABLE IF EXISTS galgame_series CASCADE;
DROP TABLE IF EXISTS galgame_link CASCADE;
DROP TABLE IF EXISTS galgame_pr CASCADE;
DROP TABLE IF EXISTS galgame_history CASCADE;
DROP TABLE IF EXISTS galgame_contributor CASCADE;

-- ──────────────────────────────────────────
-- 2. Remove metadata columns from galgame table
-- ──────────────────────────────────────────

ALTER TABLE galgame
  DROP COLUMN IF EXISTS name_en_us,
  DROP COLUMN IF EXISTS name_ja_jp,
  DROP COLUMN IF EXISTS name_zh_cn,
  DROP COLUMN IF EXISTS name_zh_tw,
  DROP COLUMN IF EXISTS banner,
  DROP COLUMN IF EXISTS intro_en_us,
  DROP COLUMN IF EXISTS intro_ja_jp,
  DROP COLUMN IF EXISTS intro_zh_cn,
  DROP COLUMN IF EXISTS intro_zh_tw,
  DROP COLUMN IF EXISTS vndb_id,
  DROP COLUMN IF EXISTS content_limit,
  DROP COLUMN IF EXISTS original_language,
  DROP COLUMN IF EXISTS age_limit,
  DROP COLUMN IF EXISTS series_id,
  DROP COLUMN IF EXISTS resource_update_time,
  DROP COLUMN IF EXISTS user_id,
  DROP COLUMN IF EXISTS status;

COMMIT;
