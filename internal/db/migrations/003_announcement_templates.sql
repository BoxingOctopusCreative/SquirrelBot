CREATE TABLE announcement_templates (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    title TEXT NOT NULL,
    body TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX announcement_templates_name ON announcement_templates (guild_id, lower(name));
