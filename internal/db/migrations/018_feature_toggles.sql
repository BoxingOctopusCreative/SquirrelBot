ALTER TABLE guilds
    ADD COLUMN welcome_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN firecall_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN announcements_enabled BOOLEAN NOT NULL DEFAULT true,
    ADD COLUMN reactions_enabled BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE log_configs
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE role_gates
    ADD COLUMN active BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE confession_configs
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE ticket_configs
    ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT true;
