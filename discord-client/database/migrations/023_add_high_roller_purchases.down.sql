-- Remove high_roller_purchase from the transaction type constraint
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_transaction_type_check 
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial', 'wager_win', 'wager_loss', 'group_wager_win', 'group_wager_loss', 'wordle_reward'));

-- Drop high_roller_purchases table
DROP TABLE IF EXISTS high_roller_purchases;