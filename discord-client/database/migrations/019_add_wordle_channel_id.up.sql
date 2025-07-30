-- Add wordle_channel_id to guild_settings table
ALTER TABLE guild_settings ADD COLUMN wordle_channel_id BIGINT;