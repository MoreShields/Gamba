-- Add missing foreign key constraints to wordle_completions and high_roller_purchases

ALTER TABLE wordle_completions
ADD CONSTRAINT fk_wordle_completions_discord_id
FOREIGN KEY (discord_id) REFERENCES users(discord_id) ON DELETE CASCADE;

ALTER TABLE high_roller_purchases
ADD CONSTRAINT fk_high_roller_purchases_discord_id
FOREIGN KEY (discord_id) REFERENCES users(discord_id) ON DELETE CASCADE;
