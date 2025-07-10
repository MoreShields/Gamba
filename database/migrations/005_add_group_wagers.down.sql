-- Drop triggers
DROP TRIGGER IF EXISTS update_group_wager_participants_updated_at ON group_wager_participants;

-- Drop indexes
DROP INDEX IF EXISTS idx_group_wager_participants_option;
DROP INDEX IF EXISTS idx_group_wager_participants_discord_id;
DROP INDEX IF EXISTS idx_group_wager_participants_group_wager;
DROP INDEX IF EXISTS idx_group_wager_options_group_wager;
DROP INDEX IF EXISTS idx_group_wagers_message;
DROP INDEX IF EXISTS idx_group_wagers_state;
DROP INDEX IF EXISTS idx_group_wagers_creator;

-- Revert balance_history constraints to previous state
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_related_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_related_type_check 
CHECK (related_type IN ('bet', 'wager'));

ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_transaction_type_check 
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'interest', 'initial', 'wager_win', 'wager_loss'));

-- Drop tables in reverse order of creation
DROP TABLE IF EXISTS group_wager_participants;
DROP TABLE IF EXISTS group_wager_options;
DROP TABLE IF EXISTS group_wagers;