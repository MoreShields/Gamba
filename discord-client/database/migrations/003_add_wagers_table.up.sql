-- Create wagers table to track public wagers between users
CREATE TABLE wagers (
    id BIGSERIAL PRIMARY KEY,
    proposer_discord_id BIGINT NOT NULL REFERENCES users(discord_id),
    target_discord_id BIGINT NOT NULL REFERENCES users(discord_id),
    amount BIGINT NOT NULL CHECK (amount > 0),
    condition TEXT NOT NULL,
    state VARCHAR(20) NOT NULL DEFAULT 'proposed' CHECK (state IN ('proposed', 'declined', 'voting', 'resolved')),
    winner_discord_id BIGINT REFERENCES users(discord_id),
    winner_balance_history_id BIGINT REFERENCES balance_history(id),
    loser_balance_history_id BIGINT REFERENCES balance_history(id),
    message_id BIGINT,
    channel_id BIGINT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    accepted_at TIMESTAMP,
    resolved_at TIMESTAMP,
    CONSTRAINT different_users CHECK (proposer_discord_id != target_discord_id),
    CONSTRAINT winner_is_participant CHECK (
        winner_discord_id IS NULL OR 
        winner_discord_id = proposer_discord_id OR 
        winner_discord_id = target_discord_id
    ),
    CONSTRAINT balance_history_when_resolved CHECK (
        (state = 'resolved' AND winner_balance_history_id IS NOT NULL AND loser_balance_history_id IS NOT NULL) OR
        (state != 'resolved' AND winner_balance_history_id IS NULL AND loser_balance_history_id IS NULL)
    )
);

-- Create wager_votes table to track participant votes
CREATE TABLE wager_votes (
    id BIGSERIAL PRIMARY KEY,
    wager_id BIGINT NOT NULL REFERENCES wagers(id) ON DELETE CASCADE,
    voter_discord_id BIGINT NOT NULL REFERENCES users(discord_id),
    vote_for_discord_id BIGINT NOT NULL REFERENCES users(discord_id),
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(wager_id, voter_discord_id)
);

-- Add related_type column to balance_history to distinguish bet vs wager references
ALTER TABLE balance_history 
ADD COLUMN related_type VARCHAR(20) CHECK (related_type IN ('bet', 'wager'));

-- Update existing bet-related balance history records
UPDATE balance_history 
SET related_type = 'bet' 
WHERE related_id IS NOT NULL 
  AND transaction_type IN ('bet_win', 'bet_loss');

-- Add new transaction types to balance_history constraint
ALTER TABLE balance_history 
DROP CONSTRAINT balance_history_transaction_type_check;

ALTER TABLE balance_history 
ADD CONSTRAINT balance_history_transaction_type_check 
CHECK (transaction_type IN ('bet_win', 'bet_loss', 'transfer_in', 'transfer_out', 'initial', 'wager_win', 'wager_loss'));

-- Indexes for efficient querying
CREATE INDEX idx_wagers_proposer ON wagers(proposer_discord_id);
CREATE INDEX idx_wagers_target ON wagers(target_discord_id);
CREATE INDEX idx_wagers_state ON wagers(state);
CREATE INDEX idx_wagers_message ON wagers(message_id, channel_id);
CREATE INDEX idx_wager_votes_wager ON wager_votes(wager_id);
CREATE INDEX idx_wager_votes_voter ON wager_votes(voter_discord_id);
CREATE INDEX idx_balance_history_related_type ON balance_history(related_type) WHERE related_type IS NOT NULL;

-- Trigger to update wager_votes.updated_at
CREATE TRIGGER update_wager_votes_updated_at BEFORE UPDATE
    ON wager_votes FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();