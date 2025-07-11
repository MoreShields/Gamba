-- Revert voting_period_minutes back to voting_period_hours
ALTER TABLE group_wagers 
DROP CONSTRAINT IF EXISTS voting_period_minutes_check;

ALTER TABLE group_wagers 
ADD COLUMN voting_period_hours INT;

-- Convert minutes back to hours (round up to nearest hour)
UPDATE group_wagers 
SET voting_period_hours = CEIL(voting_period_minutes::float / 60);

-- Make the column NOT NULL after populating data
ALTER TABLE group_wagers 
ALTER COLUMN voting_period_hours SET NOT NULL;

-- Set default value
ALTER TABLE group_wagers 
ALTER COLUMN voting_period_hours SET DEFAULT 24;

-- Drop the minutes column
ALTER TABLE group_wagers 
DROP COLUMN voting_period_minutes;