"""Control client for the mock Riot API server.

This module provides a Python client and CLI for controlling the mock server,
making it easy to simulate various game scenarios.
"""

import asyncio
from typing import Optional, Dict, Any
from enum import Enum

import httpx
import structlog
import click

logger = structlog.get_logger()


class GameState(Enum):
    """Game states matching the server."""
    NOT_IN_GAME = "not_in_game"
    IN_CHAMPION_SELECT = "in_champion_select"
    IN_GAME = "in_game"


class MockRiotControlClient:
    """Client for controlling the mock Riot API server."""
    
    def __init__(self, base_url: str = "http://localhost:8080"):
        self.base_url = base_url
        self.control_url = f"{base_url}/control"
        
    async def create_player(
        self,
        game_name: str,
        tag_line: str,
        puuid: Optional[str] = None,
        state: GameState = GameState.NOT_IN_GAME,
        queue_type_id: int = 420
    ) -> Dict[str, Any]:
        """Create a new mock player."""
        async with httpx.AsyncClient() as client:
            data = {
                "game_name": game_name,
                "tag_line": tag_line,
                "state": state.value,
                "queue_type_id": queue_type_id
            }
            if puuid:
                data["puuid"] = puuid
                
            response = await client.post(f"{self.control_url}/players", json=data)
            response.raise_for_status()
            return response.json()
            
    async def update_player_state(self, puuid: str, state: GameState) -> Dict[str, Any]:
        """Update a player's state."""
        async with httpx.AsyncClient() as client:
            response = await client.put(
                f"{self.control_url}/players/{puuid}/state",
                json={"state": state.value}
            )
            response.raise_for_status()
            return response.json()
            
    async def start_game(
        self,
        puuid: str,
        game_id: Optional[str] = None,
        queue_type_id: Optional[int] = None,
        champion_id: int = 1
    ) -> Dict[str, Any]:
        """Start a game for a player."""
        async with httpx.AsyncClient() as client:
            data = {"champion_id": champion_id}
            if game_id:
                data["game_id"] = game_id
            if queue_type_id:
                data["queue_type_id"] = queue_type_id
                
            response = await client.post(
                f"{self.control_url}/players/{puuid}/start-game",
                json=data
            )
            response.raise_for_status()
            return response.json()
            
    async def end_game(
        self,
        puuid: str,
        won: bool = True,
        duration_seconds: int = 1800,
        kills: int = 5,
        deaths: int = 3,
        assists: int = 10,
        game_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """End a game for a player with a result."""
        async with httpx.AsyncClient() as client:
            data = {
                "won": won,
                "duration_seconds": duration_seconds,
                "kills": kills,
                "deaths": deaths,
                "assists": assists
            }
            if game_id:
                data["game_id"] = game_id
                
            response = await client.post(
                f"{self.control_url}/players/{puuid}/end-game",
                json=data
            )
            response.raise_for_status()
            return response.json()
    
    async def start_tft_game(
        self,
        puuid: str,
        game_id: Optional[str] = None,
        queue_type_id: int = 1100  # Default to TFT Ranked
    ) -> Dict[str, Any]:
        """Start a TFT game for a player."""
        async with httpx.AsyncClient() as client:
            data = {"queue_type_id": queue_type_id}
            if game_id:
                data["game_id"] = game_id
                
            response = await client.post(
                f"{self.control_url}/players/{puuid}/start-tft-game",
                json=data
            )
            response.raise_for_status()
            return response.json()
    
    async def end_tft_game(
        self,
        puuid: str,
        placement: int = 4,
        duration_seconds: int = 1800,
        game_id: Optional[str] = None
    ) -> Dict[str, Any]:
        """End a TFT game for a player with a result."""
        async with httpx.AsyncClient() as client:
            data = {
                "placement": placement,
                "duration_seconds": duration_seconds
            }
            if game_id:
                data["game_id"] = game_id
                
            response = await client.post(
                f"{self.control_url}/players/{puuid}/end-tft-game",
                json=data
            )
            response.raise_for_status()
            return response.json()
            
    async def list_players(self) -> Dict[str, Any]:
        """List all mock players."""
        async with httpx.AsyncClient() as client:
            response = await client.get(f"{self.control_url}/players")
            response.raise_for_status()
            return response.json()
            
    async def delete_player(self, puuid: str) -> Dict[str, Any]:
        """Delete a mock player."""
        async with httpx.AsyncClient() as client:
            response = await client.delete(f"{self.control_url}/players/{puuid}")
            response.raise_for_status()
            return response.json()
            
    async def update_settings(
        self,
        request_delay: Optional[float] = None,
        should_return_429: Optional[bool] = None
    ) -> Dict[str, Any]:
        """Update server settings."""
        async with httpx.AsyncClient() as client:
            data = {}
            if request_delay is not None:
                data["request_delay"] = request_delay
            if should_return_429 is not None:
                data["should_return_429"] = should_return_429
                
            response = await client.put(f"{self.control_url}/settings", json=data)
            response.raise_for_status()
            return response.json()
            
    async def reset_server(self) -> Dict[str, Any]:
        """Reset server to initial state."""
        async with httpx.AsyncClient() as client:
            response = await client.post(f"{self.control_url}/reset")
            response.raise_for_status()
            return response.json()
            
    async def simulate_game_cycle(
        self,
        puuid: str,
        duration_seconds: int = 30,
        won: bool = True
    ) -> None:
        """Simulate a complete League of Legends game cycle for a player."""
        logger.info("Starting game cycle simulation", puuid=puuid)
        
        # Start game
        game_result = await self.start_game(puuid)
        logger.info("Game started", game_id=game_result["game_id"])
        
        # Wait for game duration
        logger.info(f"Simulating game for {duration_seconds} seconds...")
        await asyncio.sleep(duration_seconds)
        
        # End game
        match_result = await self.end_game(puuid, won=won, duration_seconds=duration_seconds)
        logger.info("Game ended", match_id=match_result["match_id"], won=won)
    
    async def simulate_tft_game_cycle(
        self,
        puuid: str,
        duration_seconds: int = 30,
        placement: int = 4
    ) -> None:
        """Simulate a complete TFT game cycle for a player."""
        logger.info("Starting TFT game cycle simulation", puuid=puuid)
        
        # Start TFT game
        game_result = await self.start_tft_game(puuid)
        logger.info("TFT game started", game_id=game_result["game_id"])
        
        # Wait for game duration
        logger.info(f"Simulating TFT game for {duration_seconds} seconds...")
        await asyncio.sleep(duration_seconds)
        
        # End TFT game
        match_result = await self.end_tft_game(puuid, placement=placement, duration_seconds=duration_seconds)
        logger.info("TFT game ended", match_id=match_result["match_id"], placement=placement)


# Predefined test scenarios
class TestScenarios:
    """Common test scenarios for the mock server."""
    
    @staticmethod
    async def basic_game_flow(client: MockRiotControlClient) -> str:
        """Create a player and run through a basic game flow."""
        # Create player
        player = await client.create_player("TestPlayer", "NA1")
        puuid = player["puuid"]
        
        # Start game
        await client.start_game(puuid)
        
        # Wait a bit
        await asyncio.sleep(2)
        
        # End game with win
        await client.end_game(puuid, won=True, duration_seconds=1800)
        
        return puuid
        
    @staticmethod
    async def multiple_players_concurrent_games(client: MockRiotControlClient) -> list:
        """Create multiple players with concurrent games."""
        puuids = []
        
        # Create 3 players
        for i in range(3):
            player = await client.create_player(f"Player{i}", "NA1")
            puuids.append(player["puuid"])
            
        # Start games for all players
        for i, puuid in enumerate(puuids):
            await client.start_game(puuid, champion_id=i+1)
            
        return puuids
        
    @staticmethod
    async def rapid_state_changes(client: MockRiotControlClient, puuid: str) -> None:
        """Simulate rapid state changes for testing edge cases."""
        # Rapid transitions
        await client.update_player_state(puuid, GameState.IN_CHAMPION_SELECT)
        await asyncio.sleep(0.5)
        
        await client.start_game(puuid)
        await asyncio.sleep(0.5)
        
        await client.end_game(puuid, won=False, duration_seconds=300)  # 5 min remake
        await asyncio.sleep(0.5)
        
        await client.start_game(puuid)
        await asyncio.sleep(0.5)
        
        await client.end_game(puuid, won=True, duration_seconds=2400)  # 40 min game
    
    @staticmethod
    async def basic_tft_game_flow(client: MockRiotControlClient) -> str:
        """Create a player and run through a basic TFT game flow."""
        # Create player
        player = await client.create_player("TFTPlayer", "NA1")
        puuid = player["puuid"]
        
        # Start TFT game
        await client.start_tft_game(puuid)
        
        # Wait a bit
        await asyncio.sleep(2)
        
        # End TFT game with 1st place
        await client.end_tft_game(puuid, placement=1, duration_seconds=1800)
        
        return puuid
    
    @staticmethod
    async def multiple_tft_players_concurrent_games(client: MockRiotControlClient) -> list:
        """Create multiple TFT players with concurrent games."""
        puuids = []
        
        # Create 8 players (full TFT lobby)
        for i in range(8):
            player = await client.create_player(f"TFTPlayer{i}", "NA1")
            puuids.append(player["puuid"])
            
        # Start TFT games for all players
        for puuid in puuids:
            await client.start_tft_game(puuid)
            
        return puuids


# CLI Commands
@click.group()
@click.option('--server-url', default='http://localhost:8080', help='Mock server URL')
@click.pass_context
def cli(ctx, server_url):
    """Mock Riot API Control CLI."""
    ctx.ensure_object(dict)
    ctx.obj['client'] = MockRiotControlClient(server_url)


@cli.command()
@click.argument('game_name')
@click.argument('tag_line')
@click.option('--puuid', help='Specific PUUID to use')
@click.option('--state', type=click.Choice(['not_in_game', 'in_champion_select', 'in_game']), 
              default='not_in_game')
@click.pass_context
def create_player(ctx, game_name, tag_line, puuid, state):
    """Create a new mock player."""
    client = ctx.obj['client']
    result = asyncio.run(client.create_player(
        game_name, tag_line, puuid, GameState(state)
    ))
    click.echo(f"Created player: {result}")


@cli.command()
@click.pass_context
def list_players(ctx):
    """List all mock players."""
    client = ctx.obj['client']
    result = asyncio.run(client.list_players())
    
    players = result.get('players', [])
    if not players:
        click.echo("No players found")
        return
        
    for player in players:
        click.echo(f"\n{player['game_name']}#{player['tag_line']} ({player['puuid']})")
        click.echo(f"  State: {player['state']}")
        if player['current_game_id']:
            click.echo(f"  Current Game: {player['current_game_id']}")
        if player['last_match_result']:
            click.echo(f"  Last Match: {player['last_match_result']}")


@cli.command()
@click.argument('puuid')
@click.option('--champion-id', default=1, type=int)
@click.option('--queue-type-id', type=int)
@click.pass_context
def start_game(ctx, puuid, champion_id, queue_type_id):
    """Start a League of Legends game for a player."""
    client = ctx.obj['client']
    result = asyncio.run(client.start_game(puuid, champion_id=champion_id, queue_type_id=queue_type_id))
    click.echo(f"Game started: {result}")


@cli.command()
@click.argument('puuid')
@click.option('--queue-type-id', default=1100, type=int, help='TFT queue type (1100=ranked, 1090=normal)')
@click.pass_context
def start_tft_game(ctx, puuid, queue_type_id):
    """Start a TFT game for a player."""
    client = ctx.obj['client']
    result = asyncio.run(client.start_tft_game(puuid, queue_type_id=queue_type_id))
    click.echo(f"TFT game started: {result}")


@cli.command()
@click.argument('puuid')
@click.option('--won/--lost', default=True)
@click.option('--duration', default=1800, type=int, help='Duration in seconds')
@click.option('--kills', default=5, type=int)
@click.option('--deaths', default=3, type=int)
@click.option('--assists', default=10, type=int)
@click.option('--game-id', help='Specific game ID to end (defaults to current game)')
@click.pass_context
def end_game(ctx, puuid, won, duration, kills, deaths, assists, game_id):
    """End a League of Legends game for a player."""
    client = ctx.obj['client']
    result = asyncio.run(client.end_game(
        puuid, won=won, duration_seconds=duration,
        kills=kills, deaths=deaths, assists=assists,
        game_id=game_id
    ))
    click.echo(f"Game ended: {result}")


@cli.command()
@click.argument('puuid')
@click.option('--placement', default=4, type=int, help='Final placement (1-8)')
@click.option('--duration', default=1800, type=int, help='Duration in seconds')
@click.option('--game-id', help='Specific game ID to end (defaults to current game)')
@click.pass_context
def end_tft_game(ctx, puuid, placement, duration, game_id):
    """End a TFT game for a player."""
    client = ctx.obj['client']
    result = asyncio.run(client.end_tft_game(
        puuid, placement=placement, duration_seconds=duration,
        game_id=game_id
    ))
    click.echo(f"TFT game ended: {result}")


@cli.command()
@click.argument('puuid')
@click.option('--duration', default=30, type=int, help='Game duration in seconds')
@click.option('--won/--lost', default=True)
@click.pass_context
def simulate_game(ctx, puuid, duration, won):
    """Simulate a complete League of Legends game cycle."""
    client = ctx.obj['client']
    asyncio.run(client.simulate_game_cycle(puuid, duration, won))
    click.echo("Game cycle completed")


@cli.command()
@click.argument('puuid')
@click.option('--duration', default=30, type=int, help='Game duration in seconds')
@click.option('--placement', default=4, type=int, help='Final placement (1-8)')
@click.pass_context
def simulate_tft_game(ctx, puuid, duration, placement):
    """Simulate a complete TFT game cycle."""
    client = ctx.obj['client']
    asyncio.run(client.simulate_tft_game_cycle(puuid, duration, placement))
    click.echo("TFT game cycle completed")


@cli.command()
@click.option('--delay', type=float, help='Request delay in seconds')
@click.option('--rate-limit/--no-rate-limit', default=False, help='Force 429 responses')
@click.pass_context
def settings(ctx, delay, rate_limit):
    """Update server settings."""
    client = ctx.obj['client']
    result = asyncio.run(client.update_settings(
        request_delay=delay,
        should_return_429=rate_limit
    ))
    click.echo(f"Settings updated: {result}")


@cli.command()
@click.pass_context
def reset(ctx):
    """Reset server to initial state."""
    client = ctx.obj['client']
    result = asyncio.run(client.reset_server())
    click.echo("Server reset")


@cli.command()
@click.pass_context
def run_scenario(ctx):
    """Run a basic test scenario."""
    client = ctx.obj['client']
    
    async def scenario():
        click.echo("Running basic game flow scenario...")
        puuid = await TestScenarios.basic_game_flow(client)
        click.echo(f"Scenario completed for player {puuid}")
        
    asyncio.run(scenario())


if __name__ == '__main__':
    cli()