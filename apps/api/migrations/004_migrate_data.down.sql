-- Data migration is not reversible — the source text[] columns still exist.
-- Count columns are reset to 0 by 002 down migration.
-- Relation table data is dropped by 001 down migration.
-- This file is intentionally a no-op.
SELECT 1;
