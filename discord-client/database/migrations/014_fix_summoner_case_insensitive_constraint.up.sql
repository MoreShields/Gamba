-- Fix missing case-insensitive unique constraint on summoners table
-- The repository code expects LOWER(game_name), LOWER(tag_line) constraint but it was missing

CREATE UNIQUE INDEX IF NOT EXISTS unique_summoner_tagline_ci 
ON summoners (LOWER(game_name), LOWER(tag_line));