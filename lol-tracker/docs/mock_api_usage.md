# Mock Riot API Usage Guide

## Quick Start

### 1. Start the Mock Server

```bash
# In terminal 1
cd lol-tracker
python -m lol_tracker.riot_api.mock_riot_server
```

### 2. Run lol-tracker with Mock API

```bash
# In terminal 2
export MOCK_RIOT_API_URL=http://localhost:8080
cd lol-tracker
make run
```

### 3. Control the Mock Server

```bash
# In terminal 3
cd lol-tracker

# Create a player
python -m lol_tracker.riot_api.mock_riot_control create-player TestPlayer NA1

# Start a game
python -m lol_tracker.riot_api.mock_riot_control start-game <puuid>

# End a game
python -m lol_tracker.riot_api.mock_riot_control end-game <puuid> --won
```

## Running Tests

### Automated Testing
```bash
# Run all tests with mock API
cd lol-tracker
python scripts/test_with_mock_riot.py
```

### Manual Testing Scenarios
```bash
# Run example scenarios
cd lol-tracker
python examples/mock_riot_scenarios.py
```

## For Sub-Agents

When validating changes to the lol-tracker service:

1. **Use the automated test script:**
   ```bash
   python scripts/test_with_mock_riot.py
   ```
   This will:
   - Start the mock server
   - Set up test data
   - Run integration tests
   - Validate game flows
   - Provide a pass/fail result

2. **For custom scenarios:**
   ```python
   from lol_tracker.riot_api import MockRiotControlClient
   
   client = MockRiotControlClient()
   
   # Create your test scenario
   player = await client.create_player("Test", "NA1")
   await client.start_game(player["puuid"])
   # ... etc
   ```

## Environment Variables

- `MOCK_RIOT_API_URL`: Set to use mock API instead of real Riot API
  - Example: `export MOCK_RIOT_API_URL=http://localhost:8080`
  - When not set, real Riot API is used

## Common Use Cases

### Testing State Transitions
```bash
# Rapid state changes
python -m lol_tracker.riot_api.mock_riot_control create-player RapidTest NA1
python -m lol_tracker.riot_api.mock_riot_control start-game <puuid>
sleep 5
python -m lol_tracker.riot_api.mock_riot_control end-game <puuid> --won
```

### Testing Error Handling
```bash
# Enable rate limiting
python -m lol_tracker.riot_api.mock_riot_control settings --rate-limit

# Test with non-existent player
python -m lol_tracker.riot_api.mock_riot_control start-game non-existent-puuid
```

### Testing Multiple Players
```bash
# Create multiple players
for i in {1..5}; do
  python -m lol_tracker.riot_api.mock_riot_control create-player "Player$i" NA1
done

# List all players
python -m lol_tracker.riot_api.mock_riot_control list-players
```

## Troubleshooting

- **Mock server not starting**: Check if port 8080 is already in use
- **Connection refused**: Ensure mock server is running before using control client
- **Tests failing**: Reset server state with `python -m lol_tracker.riot_api.mock_riot_control reset`