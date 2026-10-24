CREATE TABLE gate_questions (
    guild_id TEXT NOT NULL REFERENCES guilds (id) ON DELETE CASCADE,
    position INTEGER NOT NULL,
    prompt TEXT NOT NULL,
    PRIMARY KEY (guild_id, position),
    CONSTRAINT gate_questions_position CHECK (position BETWEEN 1 AND 4),
    CONSTRAINT gate_questions_prompt CHECK (char_length(prompt) BETWEEN 1 AND 45)
);
