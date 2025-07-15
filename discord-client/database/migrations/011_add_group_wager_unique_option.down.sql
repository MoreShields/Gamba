-- Remove unique constraint on group wager options
DROP INDEX IF EXISTS idx_group_wager_options_unique_text;