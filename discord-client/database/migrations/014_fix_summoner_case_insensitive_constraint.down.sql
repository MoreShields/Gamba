-- Rollback case-insensitive unique constraint on summoners table

DROP INDEX IF EXISTS unique_summoner_tagline_ci;