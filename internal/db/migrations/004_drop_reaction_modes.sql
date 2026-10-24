ALTER TABLE reaction_roles
    DROP CONSTRAINT reaction_roles_mode,
    DROP CONSTRAINT reaction_roles_max,
    DROP COLUMN mode,
    DROP COLUMN max_roles;
