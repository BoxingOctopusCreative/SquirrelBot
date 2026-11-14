CREATE TABLE confession_configs (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    review_channel_id TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL DEFAULT ''
);

CREATE TABLE confessions (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    author_id TEXT NOT NULL,
    author_name TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    review_channel_id TEXT NOT NULL DEFAULT '',
    review_message_id TEXT NOT NULL DEFAULT '',
    public_channel_id TEXT NOT NULL DEFAULT '',
    public_message_id TEXT NOT NULL DEFAULT '',
    reviewer_id TEXT NOT NULL DEFAULT '',
    reviewer_name TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    decided_at TIMESTAMPTZ,
    CONSTRAINT confessions_status CHECK (status IN ('pending', 'approved', 'rejected'))
);

CREATE INDEX confessions_pending_author ON confessions (guild_id, author_id) WHERE status = 'pending';
