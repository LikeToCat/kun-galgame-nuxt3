BEGIN;

ALTER TABLE topic_poll_option DROP COLUMN IF EXISTS vote_count;

ALTER TABLE galgame_website DROP COLUMN IF EXISTS comment_count;
ALTER TABLE galgame_website DROP COLUMN IF EXISTS favorite_count;
ALTER TABLE galgame_website DROP COLUMN IF EXISTS like_count;

ALTER TABLE galgame_rating DROP COLUMN IF EXISTS comment_count;
ALTER TABLE galgame_rating DROP COLUMN IF EXISTS like_count;

ALTER TABLE galgame_comment DROP COLUMN IF EXISTS like_count;

ALTER TABLE galgame_resource DROP COLUMN IF EXISTS like_count;

ALTER TABLE topic_reply DROP COLUMN IF EXISTS comment_count;
ALTER TABLE topic_reply DROP COLUMN IF EXISTS dislike_count;
ALTER TABLE topic_reply DROP COLUMN IF EXISTS like_count;

ALTER TABLE topic DROP COLUMN IF EXISTS upvote_count;
ALTER TABLE topic DROP COLUMN IF EXISTS favorite_count;
ALTER TABLE topic DROP COLUMN IF EXISTS comment_count;
ALTER TABLE topic DROP COLUMN IF EXISTS reply_count;
ALTER TABLE topic DROP COLUMN IF EXISTS dislike_count;
ALTER TABLE topic DROP COLUMN IF EXISTS like_count;

ALTER TABLE galgame DROP COLUMN IF EXISTS rating_count;
ALTER TABLE galgame DROP COLUMN IF EXISTS contributor_count;
ALTER TABLE galgame DROP COLUMN IF EXISTS comment_count;
ALTER TABLE galgame DROP COLUMN IF EXISTS resource_count;
ALTER TABLE galgame DROP COLUMN IF EXISTS favorite_count;
ALTER TABLE galgame DROP COLUMN IF EXISTS like_count;

COMMIT;
