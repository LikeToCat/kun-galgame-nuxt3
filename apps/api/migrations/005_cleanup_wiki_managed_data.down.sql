-- 005 down: NOT REVERSIBLE
--
-- This migration cannot be reversed. The metadata columns and 14 wiki-managed
-- tables that were dropped have had their data migrated to the wiki service.
-- Restoring them would require re-importing from the wiki service.
--
-- To recover, restore from a database backup taken before migration 005.

SELECT 'Migration 005 is not reversible. Restore from backup if needed.' AS warning;
