-- Create group_wagers table for multi-participant wagers
CREATE TABLE group_wagers (
    id BIGSERIAL PRIMARY KEY,
    creator_discord_id BIGINT NOT NULL REFERENCES users(discord_id),
    condition TEXT NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (state IN ('active', 'resolved', 'cancelled')),
    resolver_discord_id BIGINT REFERENCES users(discord_id),
    winning_option_id BIGINT, -- Will be foreign key after options table created
    total_pot BIGINT NOT NULL DEFAULT 0,
    min_participants INT NOT NULL DEFAULT 3,
    message_id BIGINT NOT NULL DEFAULT 0,
    channel_id BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    CONSTRAINT resolver_when_resolved CHECK (
        (state = 'resolved' AND resolver_discord_id IS NOT NULL) OR
        (state != 'resolved')
    )
);

-- Create group_wager_options table for possible outcomes
CREATE TABLE group_wager_options (
    id BIGSERIAL PRIMARY KEY,
    group_wager_id BIGINT NOT NULL REFERENCES group_wagers(id) ON DELETE CASCADE,
    option_text TEXT NOT NULL,
    option_order SMALLINT NOT NULL,
    total_amount BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Add foreign key constraint now that options table exists
ALTER TABLE group_wagers 
ADD CONSTRAINT fk_winning_option 
FOREIGN KEY (winning_option_id) REFERENCES group_wager_options(id);

-- Create group_wager_participants table combining participant info, choice, and amount
CREATE TABLE group_wager_participants (
    id BIGSERIAL PRIMARY KEY,
    group_wager_id BIGINT NOT NULL REFERENCES group_wagers(id) ON DELETE CASCADE,
    discord_id BIGINT NOT NULL REFERENCES users(discord_id),
    option_id BIGINT NOT NULL REFERENCES group_wager_options(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL CHECK (amount > 0),
    payout_amount BIGINT,
    balance_history_id BIGINT REFERENCES balance_history(id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(group_wager_id, discord_id),
    CONSTRAINT payout_when_resolved CHECK (
        (payout_amount IS NOT NULL AND balance_history_id IS NOT NULL) OR
        (payout_amount IS NULL AND balance_history_id IS NULL)
    )
);

-- Add new transaction types to balance_history
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_transaction_type_check 
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial', 'wager_win', 'wager_loss', 'group_wager_win', 'group_wager_loss'));

-- Update related_type constraint to include group_wager
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_related_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_related_type_check 
CHECK (related_type IN ('bet', 'wager', 'group_wager'));

-- Create indexes for efficient querying
CREATE INDEX idx_group_wagers_creator ON group_wagers(creator_discord_id);
CREATE INDEX idx_group_wagers_state ON group_wagers(state);
CREATE INDEX idx_group_wagers_message ON group_wagers(message_id, channel_id);
CREATE INDEX idx_group_wager_options_group_wager ON group_wager_options(group_wager_id);
CREATE INDEX idx_group_wager_participants_group_wager ON group_wager_participants(group_wager_id);
CREATE INDEX idx_group_wager_participants_discord_id ON group_wager_participants(discord_id);
CREATE INDEX idx_group_wager_participants_option ON group_wager_participants(option_id);

-- Trigger to update group_wager_participants.updated_at
CREATE TRIGGER update_group_wager_participants_updated_at BEFORE UPDATE
    ON group_wager_participants FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();