-- Remove wager_type and odds_multiplier support

-- Drop indexes first
DROP INDEX IF EXISTS idx_group_wagers_type;

-- Remove columns from group_wager_options table
ALTER TABLE group_wager_options 
DROP COLUMN IF EXISTS odds_multiplier;

-- Remove columns from group_wagers table
ALTER TABLE group_wagers 
DROP COLUMN IF EXISTS wager_type;