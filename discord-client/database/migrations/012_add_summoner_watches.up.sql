-- Add summoner watch functionality with normalized schema
-- Creates two tables: summoners (core summoner data) and guild_summoner_watches (many-to-many relationship)

-- Create summoners table to store unique summoner/region combinations
CREATE TABLE summoners (
    id SERIAL PRIMARY KEY,
    summoner_name VARCHAR(255) NOT NULL,
    region VARCHAR(10) NOT NULL DEFAULT 'NA1',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_summoner_region UNIQUE (summoner_name, region)
);

-- Create many-to-many relationship table linking guilds to summoners
CREATE TABLE guild_summoner_watches (
    id SERIAL PRIMARY KEY,
    guild_id BIGINT NOT NULL,
    summoner_id INTEGER NOT NULL REFERENCES summoners(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT unique_guild_summoner UNIQUE (guild_id, summoner_id)
);

-- Create indexes for efficient querying
CREATE INDEX idx_summoners_name ON summoners(summoner_name);
CREATE INDEX idx_summoners_region ON summoners(region);
CREATE INDEX idx_summoners_name_region ON summoners(summoner_name, region);
CREATE INDEX idx_guild_summoner_watches_guild_id ON guild_summoner_watches(guild_id);
CREATE INDEX idx_guild_summoner_watches_summoner_id ON guild_summoner_watches(summoner_id);

-- Add update trigger for summoners.updated_at
CREATE TRIGGER update_summoners_updated_at BEFORE UPDATE
    ON summoners FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();