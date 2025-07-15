-- Create guild_settings table for per-guild configuration
CREATE TABLE guild_settings (
    guild_id BIGINT PRIMARY KEY,
    primary_channel_id BIGINT,
    high_roller_role_id BIGINT
);

-- Create index for efficient lookups
CREATE INDEX idx_guild_settings_guild_id ON guild_settings(guild_id);