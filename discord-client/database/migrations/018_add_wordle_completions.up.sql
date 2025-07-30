-- Create wordle_completions table for tracking daily Wordle puzzle completions
CREATE TABLE wordle_completions (
    id BIGSERIAL PRIMARY KEY,
    discord_id BIGINT NOT NULL,
    guild_id BIGINT NOT NULL,
    guess_count INT NOT NULL CHECK (guess_count BETWEEN 1 AND 6),
    max_guesses INT NOT NULL CHECK (max_guesses BETWEEN 1 AND 6),
    completed_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW()
);

-- Add unique constraint for one completion per user per guild per day
CREATE UNIQUE INDEX wordle_completions_discord_id_guild_id_date_unique 
ON wordle_completions(discord_id, guild_id, (DATE(completed_at)));

-- Index for efficient streak calculation queries
CREATE INDEX idx_wordle_streak 
ON wordle_completions(discord_id, guild_id, completed_at DESC);

-- Index for checking daily completions
CREATE INDEX idx_wordle_daily_check
ON wordle_completions(guild_id, (DATE(completed_at)));