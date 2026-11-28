ALTER TABLE guilds
    ADD COLUMN purge_enabled BOOLEAN NOT NULL DEFAULT false;

CREATE TABLE channel_purges (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL,
    preset TEXT NOT NULL DEFAULT '',
    interval_seconds INTEGER NOT NULL,
    next_run_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'scheduled',
    generation BIGINT NOT NULL DEFAULT 1,
    cursor_id TEXT NOT NULL DEFAULT '',
    removed_count INTEGER NOT NULL DEFAULT 0,
    skipped_count INTEGER NOT NULL DEFAULT 0,
    last_run_at TIMESTAMPTZ,
    last_removed INTEGER NOT NULL DEFAULT 0,
    last_skipped INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, channel_id),
    CONSTRAINT channel_purges_interval CHECK (interval_seconds BETWEEN 900 AND 2592000),
    CONSTRAINT channel_purges_preset CHECK (preset IN ('', '1h', '4h', '12h', '24h', '7d')),
    CONSTRAINT channel_purges_status CHECK (status IN ('scheduled', 'running', 'missing')),
    CONSTRAINT channel_purges_counts CHECK (
        removed_count >= 0 AND skipped_count >= 0 AND last_removed >= 0 AND last_skipped >= 0
    )
);

CREATE INDEX channel_purges_due ON channel_purges (next_run_at) WHERE status = 'scheduled';
