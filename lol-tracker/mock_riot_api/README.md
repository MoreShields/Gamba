# Riot API Mock System

This directory contains the Riot API integration, including both real and mock implementations for local development and testing.

## Overview

The mock system allows you to:
- Run end-to-end tests without hitting the real Riot API
- Rapidly iterate on game state transitions
- Test edge cases and error scenarios
- Develop offline without API keys

## Usage

### 1. Using the Mock API

Set the environment variable in .env to use the mock API:

`RIOT_API_URL=http://localhost:8080`

Now when you run the lol-tracker service, it will automatically use the mock API instead of the real one.


### 2. Starting the Mock Server

note: make sure you're in the lol-tracker/ directory

```bash
# Start the mock server on default port 8080
python -m mock_riot_api.mock_riot_server
```

### 3. Controlling the Mock Server


```bash
# Create a player
python -m mock_riot_api.control create-player Feviben NA1

# List all players
python -m mock_riot_api.control list-players
```

##### League of Legends Commands

```bash
# Start a League game
python -m mock_riot_api.control start-game <puuid>

# End a League game
python -m mock_riot_api.control end-game <puuid> --won

# Simulate a complete League game cycle
python -m mock_riot_api.control simulate-game <puuid> --duration 30
```

##### TFT Commands

```bash
# Start a TFT game
python -m mock_riot_api.control start-tft-game <puuid>

# Start a TFT game with custom queue type (1100=ranked, 1090=normal)
python -m mock_riot_api.control start-tft-game <puuid> --queue-type-id 1090

# End a TFT game with placement (1-8)
python -m mock_riot_api.control end-tft-game <puuid> --placement 1 --duration 1800

# Simulate a complete TFT game cycle
python -m mock_riot_api.control simulate-tft-game <puuid> --duration 30 --placement 3
```