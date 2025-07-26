-- Revert creator_discord_id back to NOT NULL
ALTER TABLE group_wagers ALTER COLUMN creator_discord_id SET NOT NULL;