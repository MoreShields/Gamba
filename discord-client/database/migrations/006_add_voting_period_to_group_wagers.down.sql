-- Remove voting period index
DROP INDEX IF EXISTS idx_group_wagers_voting_ends_at;

-- Remove voting period consistency constraint
ALTER TABLE group_wagers 
DROP CONSTRAINT IF EXISTS voting_period_consistency;

-- Remove voting period fields
ALTER TABLE group_wagers 
DROP COLUMN IF EXISTS voting_ends_at,
DROP COLUMN IF EXISTS voting_starts_at,
DROP COLUMN IF EXISTS voting_period_hours;