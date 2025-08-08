# Gambler Discord Bot

A fun gambling game Discord bot where players can bet with custom odds.

## Development Setup

```bash
# Copy environment configuration
cp .env.example .env
# Edit .env with your Discord token and other settings

# Start all services
make dev
```

## Database Access

The development environment creates two separate databases in a single PostgreSQL container:
- `discord_db` - For the Discord bot service
- `lol_tracker_db` - For the LoL tracker service

```bash
# Access Discord bot database
make db-shell-discord

# Access LoL tracker database  
make db-shell-lol

# Drop and recreate databases
make db-drop-discord
make db-drop-lol
```

## Individual Service Commands

```bash
# Generate protobuf files
make proto

# Run tests
make test           # All tests
make test-unit      # Unit tests only
make test-integration  # Integration tests only

# Build services
make discord-client # Build Discord bot only
make lol-tracker    # Build LoL tracker only

# Clean build artifacts
make clean
```