-- Add multi-guild support by creating user_guild_accounts table
-- and adding guild_id to all relevant tables

-- Create user_guild_accounts table to track per-guild user balances
CREATE TABLE user_guild_accounts (
    id BIGSERIAL PRIMARY KEY,
    discord_id BIGINT NOT NULL REFERENCES users(discord_id) ON DELETE CASCADE,
    guild_id BIGINT NOT NULL,
    balance BIGINT NOT NULL DEFAULT 100000,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(discord_id, guild_id)
);

-- Create indexes for efficient querying
CREATE INDEX idx_user_guild_accounts_discord_id ON user_guild_accounts(discord_id);
CREATE INDEX idx_user_guild_accounts_guild_id ON user_guild_accounts(guild_id);
CREATE INDEX idx_user_guild_accounts_discord_guild ON user_guild_accounts(discord_id, guild_id);

-- Add update trigger for user_guild_accounts.updated_at
CREATE TRIGGER update_user_guild_accounts_updated_at BEFORE UPDATE
    ON user_guild_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Migrate existing user balances to user_guild_accounts
-- Using guild_id = 0 as default for existing data (to be updated based on actual guild)
INSERT INTO user_guild_accounts (discord_id, guild_id, balance, created_at, updated_at)
SELECT discord_id, 0, balance, created_at, updated_at
FROM users;

-- Add guild_id to balance_history table
ALTER TABLE balance_history ADD COLUMN guild_id BIGINT;

-- Update existing balance_history records with default guild_id
UPDATE balance_history SET guild_id = 0 WHERE guild_id IS NULL;

-- Make guild_id NOT NULL after populating existing records
ALTER TABLE balance_history ALTER COLUMN guild_id SET NOT NULL;

-- Update balance_history indexes to include guild_id
DROP INDEX idx_balance_history_discord_id_created;
CREATE INDEX idx_balance_history_discord_guild_created 
    ON balance_history(discord_id, guild_id, created_at DESC);

-- Add guild_id to bets table
ALTER TABLE bets ADD COLUMN guild_id BIGINT;
UPDATE bets SET guild_id = 0 WHERE guild_id IS NULL;
ALTER TABLE bets ALTER COLUMN guild_id SET NOT NULL;

-- Update bets indexes to include guild_id
DROP INDEX idx_bets_discord_id;
DROP INDEX idx_bets_discord_id_created_at;
CREATE INDEX idx_bets_discord_guild ON bets(discord_id, guild_id);
CREATE INDEX idx_bets_discord_guild_created ON bets(discord_id, guild_id, created_at DESC);

-- Add guild_id to wagers table
ALTER TABLE wagers ADD COLUMN guild_id BIGINT;
UPDATE wagers SET guild_id = 0 WHERE guild_id IS NULL;
ALTER TABLE wagers ALTER COLUMN guild_id SET NOT NULL;

-- Update wagers indexes to include guild_id
CREATE INDEX idx_wagers_guild ON wagers(guild_id);
CREATE INDEX idx_wagers_guild_state ON wagers(guild_id, state);

-- Add guild_id to wager_votes table
ALTER TABLE wager_votes ADD COLUMN guild_id BIGINT;

-- Update wager_votes with guild_id from their associated wager
UPDATE wager_votes wv
SET guild_id = w.guild_id
FROM wagers w
WHERE wv.wager_id = w.id;

ALTER TABLE wager_votes ALTER COLUMN guild_id SET NOT NULL;

-- Create index for wager_votes guild_id
CREATE INDEX idx_wager_votes_guild ON wager_votes(guild_id);

-- Add guild_id to group_wagers table
ALTER TABLE group_wagers ADD COLUMN guild_id BIGINT;
UPDATE group_wagers SET guild_id = 0 WHERE guild_id IS NULL;
ALTER TABLE group_wagers ALTER COLUMN guild_id SET NOT NULL;

-- Update group_wagers indexes to include guild_id
CREATE INDEX idx_group_wagers_guild ON group_wagers(guild_id);
CREATE INDEX idx_group_wagers_guild_state ON group_wagers(guild_id, state);

-- Finally, drop the balance column from users table
ALTER TABLE users DROP COLUMN balance;