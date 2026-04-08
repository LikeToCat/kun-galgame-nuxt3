-- 003: Convert text[] columns to jsonb
-- Must DROP DEFAULT before ALTER TYPE, then SET new DEFAULT.

BEGIN;

ALTER TABLE galgame_engine ALTER COLUMN alias DROP DEFAULT;
ALTER TABLE galgame_engine ALTER COLUMN alias TYPE jsonb USING to_jsonb(alias);
ALTER TABLE galgame_engine ALTER COLUMN alias SET DEFAULT '[]'::jsonb;

ALTER TABLE galgame_rating ALTER COLUMN galgame_type DROP DEFAULT;
ALTER TABLE galgame_rating ALTER COLUMN galgame_type TYPE jsonb USING to_jsonb(galgame_type);
ALTER TABLE galgame_rating ALTER COLUMN galgame_type SET DEFAULT '[]'::jsonb;

ALTER TABLE galgame_toolset ALTER COLUMN homepage DROP DEFAULT;
ALTER TABLE galgame_toolset ALTER COLUMN homepage TYPE jsonb USING to_jsonb(homepage);
ALTER TABLE galgame_toolset ALTER COLUMN homepage SET DEFAULT '[]'::jsonb;

ALTER TABLE galgame_toolset_category ALTER COLUMN alias DROP DEFAULT;
ALTER TABLE galgame_toolset_category ALTER COLUMN alias TYPE jsonb USING to_jsonb(alias);
ALTER TABLE galgame_toolset_category ALTER COLUMN alias SET DEFAULT '[]'::jsonb;

ALTER TABLE galgame_website ALTER COLUMN domain DROP DEFAULT;
ALTER TABLE galgame_website ALTER COLUMN domain TYPE jsonb USING to_jsonb(domain);
ALTER TABLE galgame_website ALTER COLUMN domain SET DEFAULT '[]'::jsonb;

COMMIT;
