# Development Setup

## Quick Start

```bash
# Copy environment configuration
cp .env.example .env
# Edit .env with your Discord token and other settings

# Start all services
make dev

# View logs
make logs              # All services
make logs SERVICE=discord   # Discord bot only
make logs SERVICE=lol       # LoL tracker only
make logs SERVICE=postgres  # Database only

# Stop all services
make down
```

## Database Access

The development environment creates two separate databases in a single PostgreSQL container:
- `discord_db` - For the Discord bot service
- `lol_tracker_db` - For the LoL tracker service

```bash
# Access Discord bot database
make db-shell SERVICE=discord

# Access LoL tracker database  
make db-shell SERVICE=lol

# Drop and recreate databases
make db-drop SERVICE=discord
make db-drop SERVICE=lol
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
make build          # Build all
make discord-client # Build Discord bot only
make lol-tracker    # Build LoL tracker only

# Clean build artifacts
make clean
```

## Migrations (Discord bot only)

```bash
# Run migrations in development
make migrate-up-dev

# Create new migration
make migrate-create NAME=add_new_feature

# Check migration status
make migrate-status-dev
```

## Architecture

- Single PostgreSQL container with multiple databases
- Discord bot (Go) with hot reload via Air
- LoL tracker (Python) with source volume mounting
- Shared network for inter-service communication
- Environment variables loaded from `.env` file