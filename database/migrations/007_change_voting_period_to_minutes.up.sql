-- Change voting_period_hours to voting_period_minutes to support minute-level granularity
ALTER TABLE group_wagers 
ADD COLUMN voting_period_minutes INT;

-- Convert existing hours to minutes (existing data)
UPDATE group_wagers 
SET voting_period_minutes = voting_period_hours * 60;

-- Make the new column NOT NULL after populating data
ALTER TABLE group_wagers 
ALTER COLUMN voting_period_minutes SET NOT NULL;

-- Set default value
ALTER TABLE group_wagers 
ALTER COLUMN voting_period_minutes SET DEFAULT 1440; -- 24 hours in minutes

-- Drop the old column
ALTER TABLE group_wagers 
DROP COLUMN voting_period_hours;

-- Update the constraint to use minutes (minimum 5 minutes, maximum 10080 minutes = 168 hours)
ALTER TABLE group_wagers
ADD CONSTRAINT voting_period_minutes_check CHECK (voting_period_minutes >= 5 AND voting_period_minutes <= 10080);