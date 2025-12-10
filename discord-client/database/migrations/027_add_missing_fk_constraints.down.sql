-- Remove foreign key constraints from wordle_completions and high_roller_purchases

ALTER TABLE wordle_completions
DROP CONSTRAINT fk_wordle_completions_discord_id;

ALTER TABLE high_roller_purchases
DROP CONSTRAINT fk_high_roller_purchases_discord_id;
