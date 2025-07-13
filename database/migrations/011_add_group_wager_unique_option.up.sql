-- First, remove any duplicate options within the same group wager (keeping the first occurrence)
DELETE FROM group_wager_options a
WHERE a.id > (
    SELECT MIN(b.id)
    FROM group_wager_options b
    WHERE a.group_wager_id = b.group_wager_id
    AND LOWER(a.option_text) = LOWER(b.option_text)
);

-- Add unique constraint to ensure options are unique within a group wager (case-insensitive)
CREATE UNIQUE INDEX idx_group_wager_options_unique_text 
ON group_wager_options (group_wager_id, LOWER(option_text));