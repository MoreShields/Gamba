-- Add summoner watch functionality with normalized schema
-- Creates two tables: summoners (core summoner data) and guild_summoner_watches (many-to-many relationship)

-- Create summoners table to store unique summoner/tag_line combinations
CREATE TABLE summoners (
    id SERIAL PRIMARY KEY,
    game_name VARCHAR(255) NOT NULL,
    tag_line VARCHAR(5) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Create many-to-many relationship table linking guilds to summoners
CREATE TABLE guild_summoner_watches (
    id SERIAL PRIMARY KEY,
    guild_id BIGINT NOT NULL,
    summoner_id INTEGER NOT NULL REFERENCES summoners(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_guild_summoner UNIQUE (guild_id, summoner_id)
);

-- Create case-insensitive unique constraint for game_name and tag_line
CREATE UNIQUE INDEX unique_summoner_tagline_ci ON summoners (LOWER(game_name), LOWER(tag_line));

-- Create indexes for efficient querying
CREATE INDEX idx_summoners_name ON summoners(LOWER(game_name));
CREATE INDEX idx_summoners_tag_line ON summoners(LOWER(tag_line));
CREATE INDEX idx_guild_summoner_watches_guild_id ON guild_summoner_watches(guild_id);
CREATE INDEX idx_guild_summoner_watches_summoner_id ON guild_summoner_watches(summoner_id);

-- Add update trigger for summoners.updated_at
CREATE TRIGGER update_summoners_updated_at BEFORE UPDATE
    ON summoners FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();