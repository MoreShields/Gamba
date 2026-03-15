CREATE TABLE staged_bets (
    id BIGSERIAL PRIMARY KEY,
    guild_id BIGINT NOT NULL,
    discord_id BIGINT NOT NULL,
    summoner_id INTEGER NOT NULL REFERENCES summoners(id) ON DELETE CASCADE,
    amount BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_staged_bet_per_user_summoner_guild UNIQUE (guild_id, discord_id, summoner_id)
);

CREATE INDEX idx_staged_bets_guild_summoner ON staged_bets(guild_id, summoner_id);
