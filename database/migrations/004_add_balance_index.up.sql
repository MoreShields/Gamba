-- Add index on users.balance for efficient high roller queries
CREATE INDEX IF NOT EXISTS idx_users_balance ON users(balance DESC);