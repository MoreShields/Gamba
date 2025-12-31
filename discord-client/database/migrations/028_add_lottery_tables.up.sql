-- Add lottery configuration columns to guild_settings
ALTER TABLE guild_settings
ADD COLUMN lotto_channel_id BIGINT,
ADD COLUMN lotto_ticket_cost BIGINT,
ADD COLUMN lotto_difficulty BIGINT;

-- Create lottery_draws table
CREATE TABLE lottery_draws (
    id BIGSERIAL PRIMARY KEY,
    guild_id BIGINT NOT NULL,
    difficulty BIGINT NOT NULL,
    ticket_cost BIGINT NOT NULL,
    winning_number BIGINT,
    draw_time TIMESTAMP NOT NULL,
    total_pot BIGINT NOT NULL DEFAULT 0,
    completed_at TIMESTAMP,
    message_id BIGINT,
    channel_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for finding current/pending draws by guild
CREATE INDEX idx_lottery_draws_guild_pending ON lottery_draws(guild_id, draw_time)
    WHERE completed_at IS NULL;

-- Index for finding draws ready to process
CREATE INDEX idx_lottery_draws_pending_time ON lottery_draws(draw_time)
    WHERE completed_at IS NULL;

-- Create lottery_tickets table
CREATE TABLE lottery_tickets (
    id BIGSERIAL PRIMARY KEY,
    draw_id BIGINT NOT NULL REFERENCES lottery_draws(id) ON DELETE CASCADE,
    guild_id BIGINT NOT NULL,
    discord_id BIGINT NOT NULL,
    ticket_number BIGINT NOT NULL,
    purchase_price BIGINT NOT NULL,
    purchased_at TIMESTAMP NOT NULL DEFAULT NOW(),
    balance_history_id BIGINT NOT NULL,
    CONSTRAINT unique_ticket_per_user_draw UNIQUE(draw_id, discord_id, ticket_number)
);

-- Index for finding tickets by user in a draw
CREATE INDEX idx_lottery_tickets_user_draw ON lottery_tickets(draw_id, discord_id);

-- Index for finding winning tickets
CREATE INDEX idx_lottery_tickets_winning ON lottery_tickets(draw_id, ticket_number);

-- Create lottery_winners junction table for tracking all winners
CREATE TABLE lottery_winners (
    id BIGSERIAL PRIMARY KEY,
    draw_id BIGINT NOT NULL REFERENCES lottery_draws(id) ON DELETE CASCADE,
    discord_id BIGINT NOT NULL,
    ticket_id BIGINT NOT NULL REFERENCES lottery_tickets(id) ON DELETE CASCADE,
    winning_amount BIGINT NOT NULL,
    balance_history_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

-- Index for finding winners by draw
CREATE INDEX idx_lottery_winners_draw_id ON lottery_winners(draw_id);

-- Index for finding user's wins
CREATE INDEX idx_lottery_winners_discord_id ON lottery_winners(discord_id);

-- Add lottery transaction types to balance_history constraint
ALTER TABLE balance_history
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history
ADD CONSTRAINT balance_history_transaction_type_check
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial', 'wager_win', 'wager_loss', 'group_wager_win', 'group_wager_loss', 'wordle_reward', 'high_roller_purchase', 'lotto_ticket', 'lotto_win'));
