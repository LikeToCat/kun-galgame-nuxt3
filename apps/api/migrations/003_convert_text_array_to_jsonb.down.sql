-- WARNING: Converting jsonb back to text[] may lose data if non-string values were stored.

BEGIN;

ALTER TABLE galgame_website
    ALTER COLUMN domain TYPE text[] USING ARRAY(SELECT jsonb_array_elements_text(domain));
ALTER TABLE galgame_website
    ALTER COLUMN domain SET DEFAULT '{}';

ALTER TABLE galgame_toolset_category
    ALTER COLUMN alias TYPE text[] USING ARRAY(SELECT jsonb_array_elements_text(alias));
ALTER TABLE galgame_toolset_category
    ALTER COLUMN alias SET DEFAULT '{}';

ALTER TABLE galgame_toolset
    ALTER COLUMN homepage TYPE text[] USING ARRAY(SELECT jsonb_array_elements_text(homepage));
ALTER TABLE galgame_toolset
    ALTER COLUMN homepage SET DEFAULT '{}';

ALTER TABLE galgame_rating
    ALTER COLUMN galgame_type TYPE text[] USING ARRAY(SELECT jsonb_array_elements_text(galgame_type));
ALTER TABLE galgame_rating
    ALTER COLUMN galgame_type SET DEFAULT '{}';

ALTER TABLE galgame_engine
    ALTER COLUMN alias TYPE text[] USING ARRAY(SELECT jsonb_array_elements_text(alias));
ALTER TABLE galgame_engine
    ALTER COLUMN alias SET DEFAULT '{}';

COMMIT;
