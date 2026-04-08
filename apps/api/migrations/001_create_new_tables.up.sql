-- 001: Create new tables required by Go backend
-- oauth_account, topic_tag, topic_tag_relation, galgame_resource_provider

BEGIN;

-- ──────────────────────────────────────────
-- 1. OAuth account (links OAuth sub to local user)
-- ──────────────────────────────────────────
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

-- ──────────────────────────────────────────
-- 2. Topic tag (replaces topic.tag text[])
-- ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS topic_tag (
    id      SERIAL PRIMARY KEY,
    name    VARCHAR(255) NOT NULL UNIQUE,
    created TIMESTAMP NOT NULL DEFAULT NOW(),
    updated TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS topic_tag_relation (
    topic_id INT NOT NULL REFERENCES topic(id) ON DELETE CASCADE,
    tag_id   INT NOT NULL REFERENCES topic_tag(id) ON DELETE CASCADE,
    created  TIMESTAMP NOT NULL DEFAULT NOW(),
    updated  TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (topic_id, tag_id)
);

-- ──────────────────────────────────────────
-- 3. Galgame resource provider (replaces galgame_resource.provider text[])
-- ──────────────────────────────────────────
CREATE TABLE IF NOT EXISTS galgame_resource_provider (
    id          SERIAL PRIMARY KEY,
    resource_id INT NOT NULL REFERENCES galgame_resource(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    created     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated     TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (resource_id, name)
);
CREATE INDEX IF NOT EXISTS idx_grp_resource_id ON galgame_resource_provider(resource_id);

COMMIT;
