#!/usr/bin/env python3
"""Test script to verify mock API integration for both game state and summoner lookups."""

import asyncio
import os
import sys

sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lol_tracker.config import Config
from lol_tracker.riot_api import create_riot_api_client, MockRiotControlClient


async def test_mock_api_integration():
    """Test that both game state and summoner lookups use the mock API."""
    
    # Ensure MOCK_RIOT_API_URL is set
    mock_url = os.environ.get('MOCK_RIOT_API_URL', 'http://localhost:8080')
    os.environ['MOCK_RIOT_API_URL'] = mock_url
    
    print(f"Using mock API at: {mock_url}")
    
    # Create control client
    control_client = MockRiotControlClient(mock_url)
    
    # Create test player in mock server
    print("\n1. Creating test player in mock server...")
    player = await control_client.create_player("TestIntegration", "NA1", puuid="test-integration-puuid")
    print(f"   Created: {player}")
    
    # Create config and riot API client
    config = Config(
        database_url="postgresql://test:test@localhost:5432",
        database_name="test",
        riot_api_key="test_key",
    )
    
    # Create API client using factory - should be mock
    print("\n2. Creating Riot API client using factory...")
    riot_client = create_riot_api_client(config)
    print(f"   Client type: {type(riot_client).__name__}")
    
    # Test summoner lookup
    print("\n3. Testing summoner lookup...")
    try:
        summoner_info = await riot_client.get_summoner_by_name("TestIntegration", "NA1")
        print(f"   ✓ Summoner lookup successful: {summoner_info.puuid}")
    except Exception as e:
        print(f"   ✗ Summoner lookup failed: {e}")
        
    # Test game state lookup
    print("\n4. Testing game state lookup (not in game)...")
    try:
        game_info = await riot_client.get_current_game_info(
            player["puuid"], 
            "TestIntegration",
            "na1"
        )
        print(f"   ✗ Expected PlayerNotInGameError but got: {game_info}")
    except Exception as e:
        if "not currently in a game" in str(e):
            print(f"   ✓ Correctly got PlayerNotInGameError")
        else:
            print(f"   ✗ Unexpected error: {e}")
            
    # Start a game
    print("\n5. Starting a game...")
    await control_client.start_game(player["puuid"])
    
    # Test game state lookup (in game)
    print("\n6. Testing game state lookup (in game)...")
    try:
        game_info = await riot_client.get_current_game_info(
            player["puuid"],
            "TestIntegration", 
            "na1"
        )
        print(f"   ✓ Game state lookup successful: Game ID {game_info.game_id}")
    except Exception as e:
        print(f"   ✗ Game state lookup failed: {e}")
        
    # Cleanup
    await riot_client.close()
    print("\n✓ All tests completed!")


if __name__ == "__main__":
    print("Make sure the mock Riot API server is running on localhost:8080")
    print("Run: python -m lol_tracker.riot_api.mock_riot_server\n")
    
    asyncio.run(test_mock_api_integration())