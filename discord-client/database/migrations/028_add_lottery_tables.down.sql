-- Remove lottery transaction types from balance_history constraint
ALTER TABLE balance_history
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history
ADD CONSTRAINT balance_history_transaction_type_check
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial', 'wager_win', 'wager_loss', 'group_wager_win', 'group_wager_loss', 'wordle_reward', 'high_roller_purchase'));

-- Drop lottery tables (order matters for foreign keys)
DROP TABLE IF EXISTS lottery_winners;
DROP TABLE IF EXISTS lottery_tickets;
DROP TABLE IF EXISTS lottery_draws;

-- Remove lottery configuration columns from guild_settings
ALTER TABLE guild_settings
DROP COLUMN IF EXISTS lotto_channel_id,
DROP COLUMN IF EXISTS lotto_ticket_cost,
DROP COLUMN IF EXISTS lotto_difficulty;
