CREATE TABLE ticket_configs (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    panel_channel_id TEXT NOT NULL DEFAULT '',
    panel_message_id TEXT NOT NULL DEFAULT '',
    category_id TEXT NOT NULL DEFAULT '',
    support_role_id TEXT NOT NULL DEFAULT '',
    log_channel_id TEXT NOT NULL DEFAULT '',
    panel_title TEXT NOT NULL DEFAULT '',
    panel_body TEXT NOT NULL DEFAULT ''
);

CREATE TABLE tickets (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    number INTEGER NOT NULL,
    opener_id TEXT NOT NULL,
    opener_name TEXT NOT NULL,
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    channel_id TEXT NOT NULL DEFAULT '',
    control_message_id TEXT NOT NULL DEFAULT '',
    claimer_id TEXT NOT NULL DEFAULT '',
    claimer_name TEXT NOT NULL DEFAULT '',
    closed_by_id TEXT NOT NULL DEFAULT '',
    closed_by_name TEXT NOT NULL DEFAULT '',
    close_reason TEXT NOT NULL DEFAULT '',
    transcript TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at TIMESTAMPTZ,
    CONSTRAINT tickets_status CHECK (status IN ('open', 'closed', 'deleted')),
    CONSTRAINT tickets_number_positive CHECK (number > 0),
    CONSTRAINT tickets_guild_number UNIQUE (guild_id, number)
);

CREATE INDEX tickets_open_opener ON tickets (guild_id, opener_id) WHERE status = 'open';
CREATE INDEX tickets_guild_created ON tickets (guild_id, created_at DESC);
