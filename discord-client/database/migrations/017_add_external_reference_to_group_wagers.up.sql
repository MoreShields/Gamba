-- Add external reference fields to group_wagers table
ALTER TABLE group_wagers 
ADD COLUMN external_id VARCHAR(255),
ADD COLUMN external_system VARCHAR(50);

-- Create unique constraint on (external_id, external_system, guild_id) tuple
-- Only enforce uniqueness when external fields are not null
-- This allows the same game to be tracked in multiple guilds
CREATE UNIQUE INDEX idx_group_wagers_external_ref 
ON group_wagers(external_id, external_system, guild_id) 
WHERE external_id IS NOT NULL AND external_system IS NOT NULL;

-- Create index on external_id for efficient lookups
CREATE INDEX idx_group_wagers_external_id 
ON group_wagers(external_id) 
WHERE external_id IS NOT NULL;

-- Drop and recreate the resolver_when_resolved constraint to allow NULL resolvers
-- This enables system-resolved house wagers (e.g., automated LoL game outcomes)
ALTER TABLE group_wagers DROP CONSTRAINT resolver_when_resolved;
ALTER TABLE group_wagers ADD CONSTRAINT resolver_when_resolved CHECK (
    state != 'resolved' OR resolver_discord_id IS NOT NULL OR wager_type = 'house'
);