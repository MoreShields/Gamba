# Gambler Discord Bot

A fun gambling game Discord bot where players can bet with custom odds.

## Quick Start

1. Copy `.env.example` to `.env` and fill in your Discord bot token and guild ID:
   ```bash
   cp .env.example .env
   ```

2. Start the development environment:
   ```bash
   make dev
   ```

3. The bot will start with hot reload enabled.

## Development

- `make dev` - Start development environment with hot reload
- `make down` - Stop all containers
- `make test` - Run tests
- `make db-shell` - Connect to PostgreSQL shell
- `make logs` - View bot logs
