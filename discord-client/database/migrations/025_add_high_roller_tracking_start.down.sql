-- Remove high roller tracking start time from guild_settings
ALTER TABLE guild_settings
DROP COLUMN high_roller_tracking_start_time;