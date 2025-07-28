#!/usr/bin/env python3
"""Example scenarios for testing with the mock Riot API.

This script demonstrates various test scenarios that can be run
against the mock Riot API server for local development and testing.
"""

import asyncio
import logging
import sys
from typing import List

# Add parent directory to path for imports
sys.path.insert(0, '..')

from lol_tracker.riot_api import MockRiotControlClient, GameState

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)


async def scenario_basic_game_flow(client: MockRiotControlClient):
    """Simulate a basic game flow from start to finish."""
    logger.info("=== Starting Basic Game Flow Scenario ===")
    
    # Create a player
    player = await client.create_player("BasicFlowPlayer", "NA1")
    puuid = player["puuid"]
    logger.info(f"Created player: {player}")
    
    # Check initial state
    players = await client.list_players()
    logger.info(f"Initial state: {players['players'][0]['state']}")
    
    # Start a game
    logger.info("Starting game...")
    game_result = await client.start_game(puuid, champion_id=99)  # Lux
    logger.info(f"Game started: {game_result}")
    
    # Simulate game duration
    logger.info("Simulating 30 second game...")
    await asyncio.sleep(30)
    
    # End game with a win
    logger.info("Ending game with a win...")
    match_result = await client.end_game(
        puuid,
        won=True,
        duration_seconds=1800,
        kills=7,
        deaths=2,
        assists=15
    )
    logger.info(f"Game ended: {match_result}")
    
    # Check final state
    players = await client.list_players()
    logger.info(f"Final state: {players['players'][0]}")
    
    return puuid


async def scenario_rapid_transitions(client: MockRiotControlClient):
    """Test rapid state transitions."""
    logger.info("\n=== Starting Rapid Transitions Scenario ===")
    
    # Create a player
    player = await client.create_player("RapidPlayer", "NA1")
    puuid = player["puuid"]
    
    # Rapid game cycles
    for i in range(3):
        logger.info(f"\n--- Rapid cycle {i + 1} ---")
        
        # Quick transition to champion select
        await client.update_player_state(puuid, GameState.IN_CHAMPION_SELECT)
        await asyncio.sleep(0.5)
        
        # Start game
        await client.start_game(puuid)
        await asyncio.sleep(1)
        
        # End game quickly (remake)
        await client.end_game(puuid, won=False, duration_seconds=180)
        await asyncio.sleep(0.5)
    
    return puuid


async def scenario_concurrent_games(client: MockRiotControlClient):
    """Simulate multiple players with concurrent games."""
    logger.info("\n=== Starting Concurrent Games Scenario ===")
    
    puuids = []
    
    # Create 5 players
    for i in range(5):
        player = await client.create_player(f"ConcurrentPlayer{i}", "NA1")
        puuids.append(player["puuid"])
        logger.info(f"Created player {i}: {player['game_name']}")
    
    # Start games for all players
    logger.info("\nStarting games for all players...")
    for i, puuid in enumerate(puuids):
        await client.start_game(puuid, champion_id=i+1)
    
    # Wait a bit
    await asyncio.sleep(10)
    
    # End games with different results
    logger.info("\nEnding games with different results...")
    for i, puuid in enumerate(puuids):
        won = i % 2 == 0  # Even indices win
        duration = 1500 + (i * 300)  # Varying game lengths
        await client.end_game(puuid, won=won, duration_seconds=duration)
        logger.info(f"Player {i} game ended: {'Win' if won else 'Loss'}, {duration}s")
    
    return puuids


async def scenario_error_conditions(client: MockRiotControlClient):
    """Test various error conditions."""
    logger.info("\n=== Starting Error Conditions Scenario ===")
    
    # Test 1: Player not found
    logger.info("\nTest 1: Attempting to start game for non-existent player...")
    try:
        await client.start_game("non-existent-puuid")
    except Exception as e:
        logger.info(f"Expected error: {e}")
    
    # Test 2: End game when not in game
    player = await client.create_player("ErrorTestPlayer", "NA1")
    puuid = player["puuid"]
    
    logger.info("\nTest 2: Attempting to end game when not in game...")
    try:
        await client.end_game(puuid)
    except Exception as e:
        logger.info(f"Expected error: {e}")
    
    # Test 3: Rate limiting
    logger.info("\nTest 3: Testing rate limiting...")
    await client.update_settings(should_return_429=True)
    
    try:
        await client.list_players()
    except Exception as e:
        logger.info(f"Expected rate limit error: {e}")
    
    # Reset rate limiting
    await client.update_settings(should_return_429=False)
    
    return puuid


async def scenario_match_history(client: MockRiotControlClient):
    """Build up match history for a player."""
    logger.info("\n=== Starting Match History Scenario ===")
    
    # Create a player
    player = await client.create_player("HistoryPlayer", "NA1")
    puuid = player["puuid"]
    
    # Play 5 games with different results
    game_configs = [
        {"won": True, "duration": 2100, "kills": 10, "deaths": 3, "assists": 8},
        {"won": False, "duration": 1800, "kills": 3, "deaths": 7, "assists": 5},
        {"won": True, "duration": 2400, "kills": 15, "deaths": 2, "assists": 20},
        {"won": True, "duration": 1200, "kills": 5, "deaths": 1, "assists": 10},
        {"won": False, "duration": 3000, "kills": 8, "deaths": 8, "assists": 12},
    ]
    
    for i, config in enumerate(game_configs):
        logger.info(f"\nGame {i + 1}:")
        await client.start_game(puuid, champion_id=i+1)
        await asyncio.sleep(2)  # Brief pause
        result = await client.end_game(puuid, **config)
        logger.info(f"Result: {config['won']=}, {config['duration']=}s, KDA: {config['kills']}/{config['deaths']}/{config['assists']}")
    
    return puuid


async def main():
    """Run all scenarios."""
    # Initialize client
    client = MockRiotControlClient("http://localhost:8080")
    
    try:
        # Reset server state
        logger.info("Resetting server state...")
        await client.reset_server()
        
        # Run scenarios
        await scenario_basic_game_flow(client)
        await scenario_rapid_transitions(client)
        await scenario_concurrent_games(client)
        await scenario_error_conditions(client)
        await scenario_match_history(client)
        
        # Final state
        logger.info("\n=== Final Server State ===")
        players = await client.list_players()
        logger.info(f"Total players: {len(players['players'])}")
        
    except Exception as e:
        logger.error(f"Error running scenarios: {e}")
        raise


if __name__ == "__main__":
    # Make sure mock server is running
    logger.info("Make sure the mock Riot API server is running on localhost:8080")
    logger.info("Run: python -m lol_tracker.riot_api.mock_riot_server")
    
    asyncio.run(main())