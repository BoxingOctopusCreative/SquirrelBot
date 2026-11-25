ALTER TABLE guilds
    ADD COLUMN leave_channel_id TEXT NOT NULL DEFAULT '',
    ADD COLUMN leave_message TEXT NOT NULL DEFAULT '';
