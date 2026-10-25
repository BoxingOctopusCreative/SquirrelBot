CREATE TABLE bot_profiles (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    nick TEXT NOT NULL DEFAULT '',
    avatar BYTEA,
    avatar_type TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
