-- Add voting period fields to group_wagers table
ALTER TABLE group_wagers 
ADD COLUMN voting_period_hours INT NOT NULL DEFAULT 24,
ADD COLUMN voting_starts_at TIMESTAMP,
ADD COLUMN voting_ends_at TIMESTAMP;

-- Add constraint to ensure voting period fields are set for active wagers
ALTER TABLE group_wagers 
ADD CONSTRAINT voting_period_consistency CHECK (
    (state = 'active' AND voting_starts_at IS NOT NULL AND voting_ends_at IS NOT NULL) OR
    (state != 'active')
);

-- Create index for efficient querying of voting period
CREATE INDEX idx_group_wagers_voting_ends_at ON group_wagers(voting_ends_at) WHERE state = 'active';