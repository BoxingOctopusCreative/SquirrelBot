ALTER TABLE reaction_roles
    ADD COLUMN mode TEXT NOT NULL DEFAULT 'normal',
    ADD COLUMN max_roles INTEGER NOT NULL DEFAULT 1,
    ADD CONSTRAINT reaction_roles_mode CHECK (mode IN ('normal', 'unique', 'verify', 'reversed', 'limit')),
    ADD CONSTRAINT reaction_roles_max CHECK (max_roles BETWEEN 1 AND 20);
