-- Drop triggers
DROP TRIGGER IF EXISTS update_wager_votes_updated_at ON wager_votes;

-- Drop indexes
DROP INDEX IF EXISTS idx_balance_history_related_type;
DROP INDEX IF EXISTS idx_wager_votes_voter;
DROP INDEX IF EXISTS idx_wager_votes_wager;
DROP INDEX IF EXISTS idx_wagers_message;
DROP INDEX IF EXISTS idx_wagers_state;
DROP INDEX IF EXISTS idx_wagers_target;
DROP INDEX IF EXISTS idx_wagers_proposer;

-- Restore original balance_history constraint
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_transaction_type_check 
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial'));

-- Remove related_type column from balance_history
ALTER TABLE balance_history 
DROP COLUMN related_type;

-- Drop tables
DROP TABLE IF EXISTS wager_votes;
DROP TABLE IF EXISTS wagers;