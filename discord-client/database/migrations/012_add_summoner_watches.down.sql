-- Drop summoner watch functionality

-- Drop indexes first
DROP INDEX IF EXISTS idx_guild_summoner_watches_summoner_id;
DROP INDEX IF EXISTS idx_guild_summoner_watches_guild_id;
DROP INDEX IF EXISTS idx_summoners_tag_line;
DROP INDEX IF EXISTS idx_summoners_name;
DROP INDEX IF EXISTS unique_summoner_tagline_ci;

-- Drop trigger
DROP TRIGGER IF EXISTS update_summoners_updated_at ON summoners;

-- Drop tables (order matters due to foreign key constraints)
DROP TABLE IF EXISTS guild_summoner_watches;
DROP TABLE IF EXISTS summoners;