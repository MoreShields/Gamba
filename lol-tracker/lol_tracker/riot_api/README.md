# Riot API Mock System

This directory contains the Riot API integration, including both real and mock implementations for local development and testing.

## Overview

The mock system allows you to:
- Run end-to-end tests without hitting the real Riot API
- Rapidly iterate on game state transitions
- Test edge cases and error scenarios
- Develop offline without API keys

## Architecture

```
riot_api/
├── __init__.py              # Package exports
├── riot_api_client.py       # Real Riot API client
├── riot_api_factory.py      # Factory for switching between real/mock
├── mock_riot_server.py      # Mock API server
└── mock_riot_control.py     # Control client for mock server
```

## Usage

### 1. Starting the Mock Server

```bash
# Start the mock server on default port 8080
python -m lol_tracker.riot_api.mock_riot_server

# Or specify a custom port
python -m lol_tracker.riot_api.mock_riot_server --port 8090
```

### 2. Using the Mock API

Set the environment variable to use the mock API:

```bash
export MOCK_RIOT_API_URL=http://localhost:8080
```

Now when you run the lol-tracker service, it will automatically use the mock API instead of the real one.

### 3. Controlling the Mock Server

#### Via Python Client

```python
from lol_tracker.riot_api import MockRiotControlClient, GameState

# Create control client
client = MockRiotControlClient("http://localhost:8080")

# Create a player
player = await client.create_player("TestPlayer", "NA1")
puuid = player["puuid"]

# Start a game
await client.start_game(puuid, champion_id=1)

# Wait some time...
await asyncio.sleep(30)

# End the game with a win
await client.end_game(puuid, won=True, duration_seconds=1800)
```

#### Via CLI

```bash
# Create a player
python -m lol_tracker.riot_api.mock_riot_control create-player TestPlayer NA1

# List all players
python -m lol_tracker.riot_api.mock_riot_control list-players

# Start a game
python -m lol_tracker.riot_api.mock_riot_control start-game <puuid>

# End a game
python -m lol_tracker.riot_api.mock_riot_control end-game <puuid> --won

# Simulate a complete game cycle
python -m lol_tracker.riot_api.mock_riot_control simulate-game <puuid> --duration 30
```

### 4. Control API Endpoints

The mock server exposes a control API for manipulating state:

- `POST /control/players` - Create a new player
- `GET /control/players` - List all players
- `PUT /control/players/{puuid}/state` - Update player state
- `POST /control/players/{puuid}/start-game` - Start a game
- `POST /control/players/{puuid}/end-game` - End a game with result
- `DELETE /control/players/{puuid}` - Delete a player
- `PUT /control/settings` - Update server settings (delays, rate limits)
- `POST /control/reset` - Reset server to initial state

## Testing Scenarios

### Basic Game Flow
```python
# Player starts not in game
# → Starts game
# → Game progresses for some time
# → Game ends with win/loss
# → Player returns to not in game
```

### Rapid State Changes
```python
# Test edge cases with rapid transitions
# → Champion select → In game → Game ends → Immediately new game
```

### Rate Limiting
```python
# Enable rate limiting
await client.update_settings(should_return_429=True)

# All subsequent requests will return 429 errors
```

### Multiple Concurrent Games
```python
# Create multiple players with active games
# Useful for testing polling performance
```

## Environment Variables

- `MOCK_RIOT_API_URL` - URL of the mock server (e.g., http://localhost:8080)
- When not set, the real Riot API is used

## Integration with Tests

The mock server can be used in integration tests:

```python
import pytest
from lol_tracker.riot_api import MockRiotControlClient

@pytest.fixture
async def mock_riot_client():
    client = MockRiotControlClient()
    # Reset server state before each test
    await client.reset_server()
    return client

async def test_game_flow(mock_riot_client):
    # Create player
    player = await mock_riot_client.create_player("Test", "NA1")
    
    # Start game
    await mock_riot_client.start_game(player["puuid"])
    
    # Your test logic here...
```

## Differences from Real API

The mock API mimics the real Riot API structure but with some simplifications:
- No real game data validation
- Simplified match history
- Instant state transitions (no real game delays unless simulated)
- All regions return data (no region-specific routing)