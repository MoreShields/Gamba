-- Revert the group_wagers state constraint to previous state
ALTER TABLE group_wagers 
DROP CONSTRAINT group_wagers_state_check;

ALTER TABLE group_wagers 
ADD CONSTRAINT group_wagers_state_check 
CHECK (state IN ('active', 'resolved', 'cancelled'));