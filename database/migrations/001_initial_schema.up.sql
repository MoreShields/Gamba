-- Create users table
CREATE TABLE IF NOT EXISTS users (
    discord_id BIGINT PRIMARY KEY,
    username TEXT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 100000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create balance history table for time-series tracking
CREATE TABLE IF NOT EXISTS balance_history (
    id SERIAL PRIMARY KEY,
    discord_id BIGINT NOT NULL REFERENCES users(discord_id) ON DELETE CASCADE,
    balance_before BIGINT NOT NULL,
    balance_after BIGINT NOT NULL,
    change_amount BIGINT NOT NULL,
    transaction_type VARCHAR(20) NOT NULL CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'interest', 'initial')),
    transaction_metadata JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create interest runs table for idempotency
CREATE TABLE IF NOT EXISTS interest_runs (
    id SERIAL PRIMARY KEY,
    run_date DATE NOT NULL UNIQUE,
    total_interest_distributed BIGINT NOT NULL,
    users_affected INTEGER NOT NULL,
    execution_summary JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_balance_history_discord_id_created 
    ON balance_history(discord_id, created_at DESC);

CREATE INDEX idx_users_updated_at 
    ON users(updated_at);

CREATE INDEX idx_balance_history_created_at
    ON balance_history(created_at DESC);

CREATE INDEX idx_balance_history_transaction_type
    ON balance_history(transaction_type);

-- Add update trigger for users.updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_users_updated_at BEFORE UPDATE
    ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();