CREATE TABLE starboards (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    channel_id TEXT NOT NULL DEFAULT '',
    threshold INTEGER NOT NULL DEFAULT 3,
    self_star BOOLEAN NOT NULL DEFAULT false,
    nsfw BOOLEAN NOT NULL DEFAULT false,
    ignored_channel_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT starboards_threshold CHECK (threshold BETWEEN 1 AND 20)
);

CREATE TABLE starboard_posts (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    source_channel_id TEXT NOT NULL,
    source_message_id TEXT NOT NULL,
    star_channel_id TEXT NOT NULL,
    star_message_id TEXT NOT NULL,
    PRIMARY KEY (guild_id, source_message_id)
);
