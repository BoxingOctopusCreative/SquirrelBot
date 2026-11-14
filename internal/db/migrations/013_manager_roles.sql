ALTER TABLE guilds
    ADD COLUMN manager_role_ids JSONB NOT NULL DEFAULT '[]'::jsonb;
