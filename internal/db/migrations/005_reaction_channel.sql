ALTER TABLE guilds
    ADD COLUMN reaction_channel_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN reaction_message_id TEXT NOT NULL DEFAULT '';
