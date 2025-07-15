-- Update the group_wagers state constraint to include pending_resolution
ALTER TABLE group_wagers 
DROP CONSTRAINT group_wagers_state_check;

ALTER TABLE group_wagers 
ADD CONSTRAINT group_wagers_state_check 
CHECK (state IN ('active', 'pending_resolution', 'resolved', 'cancelled'));