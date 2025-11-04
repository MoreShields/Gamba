-- Fix timezone handling for Wordle completions to prevent DST-related streak issues
-- This migration converts TIMESTAMP to TIMESTAMPTZ and adds a generated column for UTC date

-- Step 1: Drop existing indexes
DROP INDEX IF EXISTS wordle_completions_discord_id_guild_id_date_unique;
DROP INDEX IF EXISTS idx_wordle_streak;
DROP INDEX IF EXISTS idx_wordle_daily_check;

-- Step 2: Convert completed_at from TIMESTAMP to TIMESTAMPTZ
-- This preserves existing timestamps and interprets them as UTC
-- TIMESTAMPTZ stores everything in UTC internally
ALTER TABLE wordle_completions
ALTER COLUMN completed_at TYPE TIMESTAMPTZ
USING completed_at AT TIME ZONE 'UTC';

-- Step 3: Convert created_at to TIMESTAMPTZ for consistency
ALTER TABLE wordle_completions
ALTER COLUMN created_at TYPE TIMESTAMPTZ
USING created_at AT TIME ZONE 'UTC';

-- Step 4: Add a generated column that automatically extracts the UTC date
-- This column is automatically computed and stored whenever completed_at changes
-- It provides a consistent UTC date regardless of session timezone
ALTER TABLE wordle_completions
ADD COLUMN completed_date DATE
GENERATED ALWAYS AS ((completed_at AT TIME ZONE 'UTC')::DATE) STORED;

-- Step 5: Recreate indexes using the generated column
-- This maintains the unique constraint while properly handling timezones
CREATE UNIQUE INDEX wordle_completions_discord_id_guild_id_date_unique
ON wordle_completions(discord_id, guild_id, completed_date);

CREATE INDEX idx_wordle_streak
ON wordle_completions(discord_id, guild_id, completed_at DESC);

CREATE INDEX idx_wordle_daily_check
ON wordle_completions(guild_id, completed_date);

-- Add comments documenting the timezone handling
COMMENT ON COLUMN wordle_completions.completed_at IS 'Completion timestamp stored as TIMESTAMPTZ (UTC internally)';
COMMENT ON COLUMN wordle_completions.completed_date IS 'UTC date of completion (generated from completed_at). Used for unique constraints and daily queries.';