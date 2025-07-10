-- Drop triggers
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
DROP FUNCTION IF EXISTS update_updated_at_column();

-- Drop indexes
DROP INDEX IF EXISTS idx_balance_history_transaction_type;
DROP INDEX IF EXISTS idx_balance_history_created_at;
DROP INDEX IF EXISTS idx_users_updated_at;
DROP INDEX IF EXISTS idx_balance_history_discord_id_created;

-- Drop tables in reverse order of dependencies
DROP TABLE IF EXISTS interest_runs;
DROP TABLE IF EXISTS balance_history;
DROP TABLE IF EXISTS users;