-- 004: Migrate data from old text[] columns to new tables,
--       and initialize denormalized count columns.
-- This migration is idempotent (ON CONFLICT DO NOTHING / safe UPDATEs).

BEGIN;

-- ──────────────────────────────────────────
-- 1. topic.tag text[] → topic_tag + topic_tag_relation
-- ──────────────────────────────────────────

-- Extract distinct tags
INSERT INTO topic_tag (name)
SELECT DISTINCT unnest(tag) FROM topic
WHERE tag IS NOT NULL AND array_length(tag, 1) > 0
ON CONFLICT (name) DO NOTHING;

-- Build relations
INSERT INTO topic_tag_relation (topic_id, tag_id, created, updated)
SELECT t.id, tt.id, NOW(), NOW()
FROM topic t, unnest(t.tag) AS tag_name
JOIN topic_tag tt ON tt.name = tag_name
WHERE t.tag IS NOT NULL AND array_length(t.tag, 1) > 0
ON CONFLICT (topic_id, tag_id) DO NOTHING;

-- ──────────────────────────────────────────
-- 2. galgame_resource.provider text[] → galgame_resource_provider
-- ──────────────────────────────────────────

INSERT INTO galgame_resource_provider (resource_id, name, created, updated)
SELECT gr.id, provider_name, gr.created, gr.updated
FROM galgame_resource gr, unnest(gr.provider) AS provider_name
WHERE gr.provider IS NOT NULL AND array_length(gr.provider, 1) > 0
ON CONFLICT (resource_id, name) DO NOTHING;

-- ──────────────────────────────────────────
-- 3. Initialize count columns from actual data
-- ──────────────────────────────────────────

-- galgame counts
UPDATE galgame g SET
    like_count = (SELECT COUNT(*) FROM galgame_like WHERE galgame_id = g.id),
    favorite_count = (SELECT COUNT(*) FROM galgame_favorite WHERE galgame_id = g.id),
    resource_count = (SELECT COUNT(*) FROM galgame_resource WHERE galgame_id = g.id),
    comment_count = (SELECT COUNT(*) FROM galgame_comment WHERE galgame_id = g.id),
    contributor_count = (SELECT COUNT(*) FROM galgame_contributor WHERE galgame_id = g.id),
    rating_count = (SELECT COUNT(*) FROM galgame_rating WHERE galgame_id = g.id);

-- topic counts
UPDATE topic t SET
    like_count = (SELECT COUNT(*) FROM topic_like WHERE topic_id = t.id),
    dislike_count = (SELECT COUNT(*) FROM topic_dislike WHERE topic_id = t.id),
    reply_count = (SELECT COUNT(*) FROM topic_reply WHERE topic_id = t.id),
    comment_count = (SELECT COUNT(*) FROM topic_comment WHERE topic_id = t.id),
    favorite_count = (SELECT COUNT(*) FROM topic_favorite WHERE topic_id = t.id),
    upvote_count = (SELECT COUNT(*) FROM topic_upvote WHERE topic_id = t.id);

-- topic_reply counts
UPDATE topic_reply tr SET
    like_count = (SELECT COUNT(*) FROM topic_reply_like WHERE topic_reply_id = tr.id),
    dislike_count = (SELECT COUNT(*) FROM topic_reply_dislike WHERE topic_reply_id = tr.id),
    comment_count = (SELECT COUNT(*) FROM topic_comment WHERE topic_reply_id = tr.id);

-- galgame_resource counts
UPDATE galgame_resource gr SET
    like_count = (SELECT COUNT(*) FROM galgame_resource_like WHERE galgame_resource_id = gr.id);

-- galgame_comment counts
UPDATE galgame_comment gc SET
    like_count = (SELECT COUNT(*) FROM galgame_comment_like WHERE galgame_comment_id = gc.id);

-- galgame_rating counts
UPDATE galgame_rating gr SET
    like_count = (SELECT COUNT(*) FROM galgame_rating_like WHERE galgame_rating_id = gr.id),
    comment_count = (SELECT COUNT(*) FROM galgame_rating_comment WHERE galgame_rating_id = gr.id);

-- galgame_website counts
UPDATE galgame_website gw SET
    like_count = (SELECT COUNT(*) FROM galgame_website_like WHERE website_id = gw.id),
    favorite_count = (SELECT COUNT(*) FROM galgame_website_favorite WHERE website_id = gw.id),
    comment_count = (SELECT COUNT(*) FROM galgame_website_comment WHERE website_id = gw.id);

-- topic_poll_option counts
UPDATE topic_poll_option po SET
    vote_count = (SELECT COUNT(*) FROM topic_poll_vote WHERE option_id = po.id);

COMMIT;
