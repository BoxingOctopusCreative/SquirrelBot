CREATE TABLE level_settings (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    xp_min INTEGER NOT NULL DEFAULT 15,
    xp_max INTEGER NOT NULL DEFAULT 25,
    rate_count INTEGER NOT NULL DEFAULT 1,
    rate_seconds INTEGER NOT NULL DEFAULT 60,
    voice_xp BOOLEAN NOT NULL DEFAULT false,
    announce TEXT NOT NULL DEFAULT 'current',
    announce_channel_id TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    announce_min_level INTEGER NOT NULL DEFAULT 0,
    announce_multiple INTEGER NOT NULL DEFAULT 0,
    reward_only BOOLEAN NOT NULL DEFAULT false,
    replace_rewards BOOLEAN NOT NULL DEFAULT false,
    ignored_channel_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    ignored_role_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    backup_at TIMESTAMPTZ,
    CONSTRAINT level_xp_min CHECK (xp_min BETWEEN 15 AND 100),
    CONSTRAINT level_xp_max CHECK (xp_max BETWEEN 15 AND 100),
    CONSTRAINT level_xp_order CHECK (xp_min <= xp_max),
    CONSTRAINT level_rate_count CHECK (rate_count BETWEEN 1 AND 20),
    CONSTRAINT level_rate_seconds CHECK (rate_seconds BETWEEN 10 AND 3600),
    CONSTRAINT level_announce CHECK (announce IN ('off', 'current', 'channel', 'dm')),
    CONSTRAINT level_announce_min CHECK (announce_min_level BETWEEN 0 AND 100),
    CONSTRAINT level_announce_multiple CHECK (announce_multiple BETWEEN 0 AND 100)
);

CREATE TABLE level_rewards (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    level INTEGER NOT NULL,
    role_id TEXT NOT NULL,
    PRIMARY KEY (guild_id, level),
    CONSTRAINT level_rewards_level CHECK (level BETWEEN 1 AND 100)
);

CREATE TABLE member_levels (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL DEFAULT '',
    xp INTEGER NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT member_levels_xp CHECK (xp BETWEEN 0 AND 2000000)
);

CREATE INDEX member_levels_rank ON member_levels (guild_id, xp DESC, user_id);

CREATE TABLE level_backups (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL DEFAULT '',
    xp INTEGER NOT NULL,
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT level_backups_xp CHECK (xp BETWEEN 0 AND 2000000)
);

CREATE TABLE level_events (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL DEFAULT '',
    level INTEGER NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT level_events_level CHECK (level BETWEEN 1 AND 100)
);

CREATE INDEX level_events_guild ON level_events (guild_id, id DESC);
