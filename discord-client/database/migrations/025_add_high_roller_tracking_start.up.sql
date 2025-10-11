-- Add high roller tracking start time to guild_settings
ALTER TABLE guild_settings
ADD COLUMN high_roller_tracking_start_time TIMESTAMP;

-- Set default to current time for existing guilds that have high roller enabled
UPDATE guild_settings
SET high_roller_tracking_start_time = NOW()
WHERE high_roller_role_id IS NOT NULL;