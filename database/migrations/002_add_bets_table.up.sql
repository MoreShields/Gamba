-- Create bets table to track all gambling bets
CREATE TABLE bets (
    id BIGSERIAL PRIMARY KEY,
    discord_id BIGINT NOT NULL,
    amount BIGINT NOT NULL,
    win_probability DECIMAL(5,4) NOT NULL CHECK (win_probability > 0 AND win_probability < 1),
    won BOOLEAN NOT NULL,
    win_amount BIGINT NOT NULL,
    balance_history_id BIGINT REFERENCES balance_history(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for common queries
CREATE INDEX idx_bets_discord_id ON bets(discord_id);
CREATE INDEX idx_bets_created_at ON bets(created_at DESC);
CREATE INDEX idx_bets_discord_id_created_at ON bets(discord_id, created_at DESC);

-- Add related_id column to balance_history for foreign key references
ALTER TABLE balance_history 
ADD COLUMN related_id BIGINT;

-- Add index for related_id lookups
CREATE INDEX idx_balance_history_related_id ON balance_history(related_id);