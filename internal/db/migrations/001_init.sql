CREATE TABLE guilds (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT '',
    icon TEXT NOT NULL DEFAULT '',
    mod_log_channel_id TEXT NOT NULL DEFAULT '',
    mod_role_id TEXT NOT NULL DEFAULT '',
    admin_role_id TEXT NOT NULL DEFAULT '',
    welcome_channel_id TEXT NOT NULL DEFAULT '',
    welcome_message TEXT NOT NULL DEFAULT '',
    autorole_id TEXT NOT NULL DEFAULT '',
    prefix TEXT NOT NULL DEFAULT '!',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE mod_cases (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    case_number INTEGER NOT NULL,
    action TEXT NOT NULL,
    target_id TEXT NOT NULL,
    target_name TEXT NOT NULL DEFAULT '',
    moderator_id TEXT NOT NULL,
    moderator_name TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    duration_seconds INTEGER,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (guild_id, case_number)
);

CREATE INDEX mod_cases_guild_created ON mod_cases (guild_id, created_at DESC);

CREATE TABLE announcements (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    author_name TEXT NOT NULL DEFAULT '',
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    ping_role_id TEXT NOT NULL DEFAULT '',
    scheduled_at TIMESTAMPTZ,
    sent_at TIMESTAMPTZ,
    message_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'scheduled',
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT announcements_status CHECK (status IN ('scheduled', 'sending', 'sent', 'failed'))
);

CREATE INDEX announcements_due ON announcements (status, scheduled_at);

CREATE TABLE escalation_roles (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    role_id TEXT NOT NULL,
    tier TEXT NOT NULL,
    max_duration_seconds INTEGER NOT NULL DEFAULT 604800,
    PRIMARY KEY (guild_id, role_id),
    CONSTRAINT escalation_roles_tier CHECK (tier IN ('mod', 'admin'))
);

CREATE TABLE escalations (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL DEFAULT '',
    role_id TEXT NOT NULL,
    tier TEXT NOT NULL,
    granted_by TEXT NOT NULL,
    granted_by_name TEXT NOT NULL DEFAULT '',
    reason TEXT NOT NULL DEFAULT '',
    starts_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at TIMESTAMPTZ NOT NULL,
    ended_at TIMESTAMPTZ,
    end_reason TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'active',
    CONSTRAINT escalations_status CHECK (status IN ('active', 'expiring', 'expired', 'revoked', 'expire_failed'))
);

CREATE INDEX escalations_due ON escalations (status, expires_at);
CREATE INDEX escalations_guild_active ON escalations (guild_id, status);

CREATE TABLE role_gates (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    required_role_id TEXT NOT NULL DEFAULT '',
    deadline_seconds INTEGER NOT NULL DEFAULT 86400,
    warn_before_seconds INTEGER NOT NULL DEFAULT 3600,
    kick_reason TEXT NOT NULL DEFAULT 'Did not receive the required role in time',
    dm_message TEXT NOT NULL DEFAULT '',
    log_channel_id TEXT NOT NULL DEFAULT '',
    exempt_role_ids JSONB NOT NULL DEFAULT '[]'::jsonb
);

CREATE TABLE gate_members (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    user_id TEXT NOT NULL,
    user_name TEXT NOT NULL DEFAULT '',
    joined_at TIMESTAMPTZ NOT NULL,
    deadline_at TIMESTAMPTZ NOT NULL,
    warned_at TIMESTAMPTZ,
    status TEXT NOT NULL DEFAULT 'pending',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT gate_members_status CHECK (status IN ('pending', 'cleared', 'kicking', 'kicked', 'kick_failed', 'exempt', 'left'))
);

CREATE INDEX gate_members_due ON gate_members (status, deadline_at);

CREATE TABLE automod_configs (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    enabled BOOLEAN NOT NULL DEFAULT false,
    block_invites BOOLEAN NOT NULL DEFAULT true,
    block_spam BOOLEAN NOT NULL DEFAULT true,
    mention_limit INTEGER NOT NULL DEFAULT 5,
    caps_percent INTEGER NOT NULL DEFAULT 70,
    caps_min_length INTEGER NOT NULL DEFAULT 12,
    bad_words JSONB NOT NULL DEFAULT '[]'::jsonb,
    action TEXT NOT NULL DEFAULT 'delete',
    timeout_seconds INTEGER NOT NULL DEFAULT 600,
    ignored_channel_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    ignored_role_ids JSONB NOT NULL DEFAULT '[]'::jsonb,
    CONSTRAINT automod_action CHECK (action IN ('delete', 'warn', 'timeout', 'kick'))
);

CREATE TABLE log_configs (
    guild_id TEXT PRIMARY KEY REFERENCES guilds (id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL DEFAULT '',
    message_delete BOOLEAN NOT NULL DEFAULT true,
    message_edit BOOLEAN NOT NULL DEFAULT true,
    member_join BOOLEAN NOT NULL DEFAULT true,
    member_leave BOOLEAN NOT NULL DEFAULT true,
    member_ban BOOLEAN NOT NULL DEFAULT true
);

CREATE TABLE reaction_roles (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    emoji TEXT NOT NULL,
    role_id TEXT NOT NULL,
    UNIQUE (message_id, emoji)
);

CREATE TABLE custom_commands (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    response TEXT NOT NULL,
    PRIMARY KEY (guild_id, name)
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL,
    username TEXT NOT NULL DEFAULT '',
    avatar TEXT NOT NULL DEFAULT '',
    access_token TEXT NOT NULL,
    refresh_token TEXT NOT NULL DEFAULT '',
    token_expires_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX sessions_expires ON sessions (expires_at);
