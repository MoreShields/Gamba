-- Add indexes to optimize scoreboard queries
-- These indexes are tailored to the specific query patterns used in GetScoreboardData

-- Index for balance_history aggregations (volume and donations)
-- Query filters by guild_id first, then groups by discord_id
CREATE INDEX IF NOT EXISTS idx_balance_history_guild_discord 
ON balance_history(guild_id, discord_id, transaction_type, change_amount);

-- Indexes for wager queries
-- For the CTE aggregation query that filters by guild and state
CREATE INDEX IF NOT EXISTS idx_wagers_guild_state 
ON wagers(guild_id, state);

-- For available balance calculation with OR condition on proposer/target
CREATE INDEX IF NOT EXISTS idx_wagers_proposer_guild_state 
ON wagers(proposer_discord_id, guild_id, state);

CREATE INDEX IF NOT EXISTS idx_wagers_target_guild_state 
ON wagers(target_discord_id, guild_id, state);

-- Index for bets queries (guild_id first since that's the primary filter)
CREATE INDEX IF NOT EXISTS idx_bets_guild_discord_won 
ON bets(guild_id, discord_id, won);

-- Index for group wager participants (unchanged - already optimal)
CREATE INDEX IF NOT EXISTS idx_group_wager_participants_scoreboard 
ON group_wager_participants(discord_id, group_wager_id);

-- Index for group wagers state queries (unchanged - already optimal)
CREATE INDEX IF NOT EXISTS idx_group_wagers_state 
ON group_wagers(guild_id, state);