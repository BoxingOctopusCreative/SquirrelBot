ALTER TABLE custom_commands
    ADD COLUMN kind TEXT NOT NULL DEFAULT 'text' CHECK (kind IN ('text', 'lua')),
    ADD COLUMN role_ids TEXT[] NOT NULL DEFAULT '{}';
