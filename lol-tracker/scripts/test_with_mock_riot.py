#!/usr/bin/env python3
"""Automated testing script using the mock Riot API.

This script is designed to be used by sub-agents or CI/CD pipelines
to validate changes to the lol-tracker service using the mock API.
"""

import asyncio
import os
import subprocess
import sys
import time
import signal
from typing import Optional
import httpx

# Add parent directory to path
sys.path.insert(0, os.path.dirname(os.path.dirname(os.path.abspath(__file__))))

from lol_tracker.riot_api import MockRiotControlClient


class MockRiotTestRunner:
    """Manages the mock Riot API server and runs tests against it."""
    
    def __init__(self, mock_port: int = 8080):
        self.mock_port = mock_port
        self.mock_url = f"http://localhost:{mock_port}"
        self.mock_process: Optional[subprocess.Popen] = None
        self.control_client = MockRiotControlClient(self.mock_url)
        
    async def start_mock_server(self) -> bool:
        """Start the mock Riot API server."""
        print(f"Starting mock Riot API server on port {self.mock_port}...")
        
        # Start the mock server process
        self.mock_process = subprocess.Popen(
            [sys.executable, "-m", "lol_tracker.riot_api.mock_riot_server", "--port", str(self.mock_port)],
            stdout=subprocess.PIPE,
            stderr=subprocess.PIPE,
            preexec_fn=os.setsid if sys.platform != 'win32' else None
        )
        
        # Wait for server to be ready
        max_retries = 30
        for i in range(max_retries):
            try:
                async with httpx.AsyncClient() as client:
                    response = await client.get(f"{self.mock_url}/control/players")
                    if response.status_code == 200:
                        print("Mock server is ready!")
                        return True
            except (httpx.ConnectError, httpx.ReadTimeout):
                pass
            
            if i < max_retries - 1:
                await asyncio.sleep(1)
        
        print("Failed to start mock server!")
        return False
        
    def stop_mock_server(self):
        """Stop the mock Riot API server."""
        if self.mock_process:
            print("Stopping mock server...")
            if sys.platform == 'win32':
                self.mock_process.terminate()
            else:
                os.killpg(os.getpgid(self.mock_process.pid), signal.SIGTERM)
            self.mock_process.wait()
            self.mock_process = None
            
    async def setup_test_data(self):
        """Set up initial test data in the mock server."""
        print("Setting up test data...")
        
        # Create some test players
        players = [
            ("TestPlayer1", "NA1"),
            ("TestPlayer2", "EUW"),
            ("TestPlayer3", "KR"),
        ]
        
        created_players = []
        for game_name, tag_line in players:
            player = await self.control_client.create_player(game_name, tag_line)
            created_players.append(player)
            print(f"Created player: {game_name}#{tag_line} (PUUID: {player['puuid']})")
            
        # Start a game for the first player
        await self.control_client.start_game(created_players[0]["puuid"])
        print(f"Started game for {created_players[0]['game_name']}")
        
        return created_players
        
    async def run_lol_tracker_tests(self):
        """Run the lol-tracker service tests with mock API."""
        print("\nRunning lol-tracker tests with mock API...")
        
        # Set environment variable to use mock API
        env = os.environ.copy()
        env["MOCK_RIOT_API_URL"] = self.mock_url
        
        # Run tests
        result = subprocess.run(
            ["make", "test-integration"],
            cwd=os.path.dirname(os.path.dirname(os.path.abspath(__file__))),
            env=env,
            capture_output=True,
            text=True
        )
        
        print("\n--- Test Output ---")
        print(result.stdout)
        if result.stderr:
            print("\n--- Test Errors ---")
            print(result.stderr)
            
        return result.returncode == 0
        
    async def validate_game_flow(self):
        """Validate a complete game flow works correctly."""
        print("\nValidating game flow...")
        
        # Create a player
        player = await self.control_client.create_player("FlowTestPlayer", "NA1")
        puuid = player["puuid"]
        
        # Start a game
        game_result = await self.control_client.start_game(puuid)
        print(f"Started game: {game_result['game_id']}")
        
        # Simulate game duration
        await asyncio.sleep(5)
        
        # End game
        match_result = await self.control_client.end_game(
            puuid,
            won=True,
            duration_seconds=1800
        )
        print(f"Ended game: {match_result['match_id']}")
        
        # Verify player state
        players = await self.control_client.list_players()
        for p in players["players"]:
            if p["puuid"] == puuid:
                if p["state"] == "not_in_game" and p["last_match_result"]:
                    print("✓ Game flow validation passed!")
                    return True
                    
        print("✗ Game flow validation failed!")
        return False
        
    async def run_all_tests(self) -> bool:
        """Run all tests and return success status."""
        try:
            # Start mock server
            if not await self.start_mock_server():
                return False
                
            # Set up test data
            await self.setup_test_data()
            
            # Run validation tests
            flow_passed = await self.validate_game_flow()
            
            # Run integration tests
            tests_passed = await self.run_lol_tracker_tests()
            
            # Summary
            print("\n=== Test Summary ===")
            print(f"Game flow validation: {'✓ PASSED' if flow_passed else '✗ FAILED'}")
            print(f"Integration tests: {'✓ PASSED' if tests_passed else '✗ FAILED'}")
            
            return flow_passed and tests_passed
            
        finally:
            self.stop_mock_server()


async def main():
    """Main entry point."""
    runner = MockRiotTestRunner()
    success = await runner.run_all_tests()
    
    sys.exit(0 if success else 1)


if __name__ == "__main__":
    asyncio.run(main())