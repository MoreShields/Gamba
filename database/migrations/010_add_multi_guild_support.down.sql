-- Revert multi-guild support changes
-- WARNING: This will combine all guild balances for each user into a single balance

-- Add balance column back to users table
ALTER TABLE users ADD COLUMN balance BIGINT NOT NULL DEFAULT 100000;

-- Restore user balances by summing across all guilds (or taking max balance)
-- Using MAX to avoid losing funds in case of multiple guild accounts
UPDATE users u
SET balance = COALESCE((
    SELECT MAX(uga.balance)
    FROM user_guild_accounts uga
    WHERE uga.discord_id = u.discord_id
), 100000);

-- Drop guild_id from group_wagers table
DROP INDEX idx_group_wagers_guild;
DROP INDEX idx_group_wagers_guild_state;
ALTER TABLE group_wagers DROP COLUMN guild_id;

-- Drop guild_id from wager_votes table
DROP INDEX idx_wager_votes_guild;
ALTER TABLE wager_votes DROP COLUMN guild_id;

-- Drop guild_id from wagers table
DROP INDEX idx_wagers_guild;
DROP INDEX idx_wagers_guild_state;
ALTER TABLE wagers DROP COLUMN guild_id;

-- Drop guild_id from bets table
DROP INDEX idx_bets_discord_guild;
DROP INDEX idx_bets_discord_guild_created;
ALTER TABLE bets DROP COLUMN guild_id;

-- Restore original bets indexes
CREATE INDEX idx_bets_discord_id ON bets(discord_id);
CREATE INDEX idx_bets_discord_id_created_at ON bets(discord_id, created_at DESC);

-- Drop guild_id from balance_history table
DROP INDEX idx_balance_history_discord_guild_created;
ALTER TABLE balance_history DROP COLUMN guild_id;

-- Restore original balance_history index
CREATE INDEX idx_balance_history_discord_id_created 
    ON balance_history(discord_id, created_at DESC);

-- Drop the user_guild_accounts table
DROP TABLE user_guild_accounts;