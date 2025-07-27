-- Remove external reference fields from group_wagers table
DROP INDEX IF EXISTS idx_group_wagers_external_id;
DROP INDEX IF EXISTS idx_group_wagers_external_ref;

-- Restore the original resolver_when_resolved constraint
ALTER TABLE group_wagers DROP CONSTRAINT IF EXISTS resolver_when_resolved;
ALTER TABLE group_wagers ADD CONSTRAINT resolver_when_resolved CHECK (
    (state = 'resolved' AND resolver_discord_id IS NOT NULL) OR
    (state != 'resolved')
);

ALTER TABLE group_wagers 
DROP COLUMN IF EXISTS external_system,
DROP COLUMN IF EXISTS external_id;