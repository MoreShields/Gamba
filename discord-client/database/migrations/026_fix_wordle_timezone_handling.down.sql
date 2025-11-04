-- Rollback timezone handling changes for Wordle completions
-- WARNING: This may reintroduce DST-related issues

-- Remove comments
COMMENT ON COLUMN wordle_completions.completed_at IS NULL;
COMMENT ON COLUMN wordle_completions.completed_date IS NULL;

-- Drop indexes
DROP INDEX IF EXISTS wordle_completions_discord_id_guild_id_date_unique;
DROP INDEX IF EXISTS idx_wordle_streak;
DROP INDEX IF EXISTS idx_wordle_daily_check;

-- Drop the generated column
ALTER TABLE wordle_completions
DROP COLUMN completed_date;

-- Convert columns back to TIMESTAMP (without timezone)
ALTER TABLE wordle_completions
ALTER COLUMN completed_at TYPE TIMESTAMP
USING completed_at AT TIME ZONE 'UTC';

ALTER TABLE wordle_completions
ALTER COLUMN created_at TYPE TIMESTAMP
USING created_at AT TIME ZONE 'UTC';

-- Recreate original indexes
CREATE UNIQUE INDEX wordle_completions_discord_id_guild_id_date_unique
ON wordle_completions(discord_id, guild_id, (DATE(completed_at)));

CREATE INDEX idx_wordle_streak
ON wordle_completions(discord_id, guild_id, completed_at DESC);

CREATE INDEX idx_wordle_daily_check
ON wordle_completions(guild_id, (DATE(completed_at)));