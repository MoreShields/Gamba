-- Add wager_type and odds_multiplier support for house vs pool wagers

-- Add wager_type column to group_wagers table
ALTER TABLE group_wagers 
ADD COLUMN wager_type VARCHAR(10) NOT NULL DEFAULT 'pool' 
CHECK (wager_type IN ('pool', 'house'));

-- Add odds_multiplier column to group_wager_options table
ALTER TABLE group_wager_options 
ADD COLUMN odds_multiplier DECIMAL(10,2) NOT NULL DEFAULT 0.0;

-- Create index for efficient querying by wager type
CREATE INDEX idx_group_wagers_type ON group_wagers(wager_type);