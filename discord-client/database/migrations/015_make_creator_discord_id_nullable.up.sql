-- Make creator_discord_id nullable to support automatic wager creation
ALTER TABLE group_wagers ALTER COLUMN creator_discord_id DROP NOT NULL;