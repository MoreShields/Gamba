-- Remove scoreboard performance indexes
DROP INDEX IF EXISTS idx_balance_history_guild_discord;
DROP INDEX IF EXISTS idx_wagers_guild_state;
DROP INDEX IF EXISTS idx_wagers_proposer_guild_state;
DROP INDEX IF EXISTS idx_wagers_target_guild_state;
DROP INDEX IF EXISTS idx_bets_guild_discord_won;
DROP INDEX IF EXISTS idx_group_wager_participants_scoreboard;
DROP INDEX IF EXISTS idx_group_wagers_state;