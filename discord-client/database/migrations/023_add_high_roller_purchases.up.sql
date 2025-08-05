-- Create high_roller_purchases table for tracking role purchases
CREATE TABLE high_roller_purchases (
    id BIGSERIAL PRIMARY KEY,
    guild_id BIGINT NOT NULL,
    discord_id BIGINT NOT NULL,
    purchase_price BIGINT NOT NULL,
    purchased_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for efficient lookup of current high roller by guild
CREATE INDEX idx_high_roller_purchases_guild_current ON high_roller_purchases(guild_id, purchased_at DESC);

-- Add high_roller_purchase to the transaction type constraint
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_transaction_type_check 
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial', 'wager_win', 'wager_loss', 'group_wager_win', 'group_wager_loss', 'wordle_reward', 'high_roller_purchase'));