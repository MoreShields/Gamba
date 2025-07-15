-- Drop indexes
DROP INDEX IF EXISTS idx_balance_history_related_id;

-- Remove related_id column from balance_history
ALTER TABLE balance_history 
DROP COLUMN IF EXISTS related_id;

-- Drop bets table and its indexes (CASCADE will drop indexes automatically)
DROP TABLE IF EXISTS bets;