-- 008: kungal-side state for the wiki investigation message stream.
--
-- See docs/galgame_wiki/08-messages.md §已读状态: each consumer (kungal /
-- moyu / admin UI) stores its own "read up to" marker because the same
-- message can be displayed on multiple sites with independent read state.
--
-- The cron cursor for ingesting /galgame/messages/feed lives in Redis
-- (key `wiki:msg:cron:since`) rather than here — it's a single
-- server-wide counter and pure cache state.

BEGIN;

CREATE TABLE IF NOT EXISTS wiki_message_read_state (
    user_id              INT PRIMARY KEY,
    last_read_message_id BIGINT NOT NULL DEFAULT 0,
    updated_at           TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

COMMIT;
