"""Mock Riot API server for local development and testing.

This module provides a fully functional mock of the Riot Games API that can be
controlled via a REST interface to simulate various game states and transitions.
"""

import asyncio
import time
from enum import Enum
from typing import Dict, Any, Optional, List
from dataclasses import dataclass
import uuid

from aiohttp import web
import structlog

logger = structlog.get_logger()


class PlayerGameState(Enum):
    """Possible game states for a player."""
    NOT_IN_GAME = "not_in_game"
    IN_CHAMPION_SELECT = "in_champion_select"
    IN_GAME = "in_game"


@dataclass
class MockPlayer:
    """Mock player data."""
    puuid: str
    game_name: str
    tag_line: str
    state: PlayerGameState = PlayerGameState.NOT_IN_GAME
    current_game_id: Optional[str] = None
    current_game_start_time: Optional[int] = None
    current_champion_id: Optional[int] = None
    last_match_result: Optional[Dict[str, Any]] = None
    queue_type_id: int = 420  # Default to ranked solo/duo


@dataclass
class MockGameInfo:
    """Mock game information."""
    game_id: str
    game_type: str = "MATCHED_GAME"
    game_start_time: Optional[int] = None
    map_id: int = 11  # Summoner's Rift
    game_length: int = 0
    platform_id: str = "NA1"
    game_mode: str = "CLASSIC"
    game_queue_config_id: int = 420
    participants: Optional[List[Dict[str, Any]]] = None

    def __post_init__(self):
        if self.game_start_time is None:
            self.game_start_time = int(time.time() * 1000)
        if self.participants is None:
            self.participants = []

    def to_api_response(self) -> Dict[str, Any]:
        """Convert to Riot API response format."""
        return {
            "gameId": int(self.game_id),
            "gameType": self.game_type,
            "gameStartTime": self.game_start_time,
            "mapId": self.map_id,
            "gameLength": self.game_length,
            "platformId": self.platform_id,
            "gameMode": self.game_mode,
            "gameQueueConfigId": self.game_queue_config_id,
            "participants": self.participants
        }


@dataclass 
class MockMatchResult:
    """Mock match result data."""
    match_id: str
    game_creation: int
    game_duration: int
    game_end_timestamp: int
    participants: List[Dict[str, Any]]
    queue_id: int = 420

    def to_api_response(self) -> Dict[str, Any]:
        """Convert to Riot API match response format."""
        return {
            "metadata": {
                "matchId": self.match_id,
                "participants": [p["puuid"] for p in self.participants]
            },
            "info": {
                "gameCreation": self.game_creation,
                "gameDuration": self.game_duration,
                "gameEndTimestamp": self.game_end_timestamp,
                "gameMode": "CLASSIC",
                "gameType": "MATCHED_GAME",
                "mapId": 11,
                "platformId": "NA1",
                "queueId": self.queue_id,
                "participants": self.participants
            }
        }


class MockRiotAPIServer:
    """Mock Riot API server with control endpoints."""
    
    def __init__(self, port: int = 8080):
        self.port = port
        self.app = web.Application()
        self.players: Dict[str, MockPlayer] = {}
        self.games: Dict[str, MockGameInfo] = {}
        self.match_results: Dict[str, MockMatchResult] = {}
        self.request_delay: float = 0  # Configurable delay for rate limit testing
        self.should_return_429: bool = False  # Force rate limit errors
        self.setup_routes()
        
    def setup_routes(self):
        """Set up all API routes."""
        # Riot API endpoints
        self.app.router.add_get('/riot/account/v1/accounts/by-riot-id/{game_name}/{tag_line}', self.get_account_by_riot_id)
        self.app.router.add_get('/lol/spectator/v5/active-games/by-summoner/{puuid}', self.get_current_game)
        self.app.router.add_get('/lol/match/v5/matches/{match_id}', self.get_match_info)
        
        # TFT endpoints
        self.app.router.add_get('/lol/spectator/tft/v5/active-games/by-puuid/{puuid}', self.get_current_tft_game)
        self.app.router.add_get('/tft/match/v1/matches/{match_id}', self.get_tft_match_info)
        
        # Control endpoints
        self.app.router.add_post('/control/players', self.create_player)
        self.app.router.add_put('/control/players/{puuid}/state', self.update_player_state)
        self.app.router.add_post('/control/players/{puuid}/start-game', self.start_game)
        self.app.router.add_post('/control/players/{puuid}/end-game', self.end_game)
        self.app.router.add_post('/control/players/{puuid}/start-tft-game', self.start_tft_game)
        self.app.router.add_post('/control/players/{puuid}/end-tft-game', self.end_tft_game)
        self.app.router.add_get('/control/players', self.list_players)
        self.app.router.add_delete('/control/players/{puuid}', self.delete_player)
        self.app.router.add_put('/control/settings', self.update_settings)
        self.app.router.add_post('/control/reset', self.reset_server)
        
    async def apply_request_delay(self):
        """Apply configured request delay for rate limit simulation."""
        if self.request_delay > 0:
            await asyncio.sleep(self.request_delay)
            
    async def check_rate_limit(self) -> Optional[web.Response]:
        """Check if we should return a rate limit error."""
        if self.should_return_429:
            return web.json_response(
                {"status": {"message": "Rate limit exceeded", "status_code": 429}},
                status=429,
                headers={"Retry-After": "60"}
            )
        return None
        
    # Riot API endpoints
    async def get_account_by_riot_id(self, request: web.Request) -> web.Response:
        """Mock /riot/account/v1/accounts/by-riot-id endpoint."""
        await self.apply_request_delay()
        
        if rate_limit_response := await self.check_rate_limit():
            return rate_limit_response
            
        game_name = request.match_info['game_name']
        tag_line = request.match_info['tag_line']
        
        # Find player by game name and tag line
        for player in self.players.values():
            if player.game_name == game_name and player.tag_line == tag_line:
                return web.json_response({
                    "puuid": player.puuid,
                    "gameName": player.game_name,
                    "tagLine": player.tag_line
                })
                
        return web.json_response(
            {"status": {"message": "Account not found", "status_code": 404}},
            status=404
        )
        
    async def get_current_game(self, request: web.Request) -> web.Response:
        """Mock /lol/spectator/v5/active-games/by-summoner endpoint."""
        await self.apply_request_delay()
        
        if rate_limit_response := await self.check_rate_limit():
            return rate_limit_response
            
        puuid = request.match_info['puuid']
        
        if puuid not in self.players:
            return web.json_response(
                {"status": {"message": "Player not found", "status_code": 404}},
                status=404
            )
            
        player = self.players[puuid]
        
        if player.state == PlayerGameState.NOT_IN_GAME:
            return web.json_response(
                {"status": {"message": "Player is not currently in a game", "status_code": 404}},
                status=404
            )
            
        # Return current game info
        if player.current_game_id and player.current_game_id in self.games:
            game = self.games[player.current_game_id]
            # Update game length based on current time
            if game.game_start_time:
                game.game_length = int((time.time() * 1000 - game.game_start_time) / 1000)
            return web.json_response(game.to_api_response())
            
        # Create a default game if none exists
        game_id = str(uuid.uuid4().int)[:10]
        game = MockGameInfo(
            game_id=game_id,
            game_queue_config_id=player.queue_type_id,
            participants=[{
                "puuid": player.puuid,
                "summonerName": player.game_name,
                "championId": player.current_champion_id or 1,
                "teamId": 100
            }]
        )
        self.games[game_id] = game
        player.current_game_id = game_id
        player.current_game_start_time = game.game_start_time
        
        return web.json_response(game.to_api_response())
        
    async def get_match_info(self, request: web.Request) -> web.Response:
        """Mock /lol/match/v5/matches endpoint."""
        await self.apply_request_delay()
        
        if rate_limit_response := await self.check_rate_limit():
            return rate_limit_response
            
        match_id = request.match_info['match_id']
        
        if match_id in self.match_results:
            return web.json_response(self.match_results[match_id].to_api_response())
            
        return web.json_response(
            {"status": {"message": "Match not found", "status_code": 404}},
            status=404
        )
    
        
    # Control endpoints
    async def create_player(self, request: web.Request) -> web.Response:
        """Create a new mock player."""
        data = await request.json()
        
        puuid = data.get("puuid", str(uuid.uuid4()))
        player = MockPlayer(
            puuid=puuid,
            game_name=data["game_name"],
            tag_line=data["tag_line"],
            state=PlayerGameState(data.get("state", "not_in_game")),
            queue_type_id=data.get("queue_type_id", 420)
        )
        
        self.players[puuid] = player
        
        logger.info("Created mock player", puuid=puuid, game_name=player.game_name)
        
        return web.json_response({
            "puuid": player.puuid,
            "game_name": player.game_name,
            "tag_line": player.tag_line,
            "state": player.state.value
        })
        
    async def update_player_state(self, request: web.Request) -> web.Response:
        """Update a player's state."""
        puuid = request.match_info['puuid']
        data = await request.json()
        
        if puuid not in self.players:
            return web.json_response({"error": "Player not found"}, status=404)
            
        player = self.players[puuid]
        new_state = PlayerGameState(data["state"])
        
        logger.info("Updating player state", 
                   puuid=puuid, 
                   old_state=player.state.value,
                   new_state=new_state.value)
        
        player.state = new_state
        
        # Clear game data if transitioning to NOT_IN_GAME
        if new_state == PlayerGameState.NOT_IN_GAME:
            player.current_game_id = None
            player.current_game_start_time = None
            player.current_champion_id = None
            
        return web.json_response({"status": "updated", "state": player.state.value})
        
    async def start_game(self, request: web.Request) -> web.Response:
        """Start a game for a player."""
        puuid = request.match_info['puuid']
        data = await request.json()
        
        if puuid not in self.players:
            return web.json_response({"error": "Player not found"}, status=404)
            
        player = self.players[puuid]
        
        # Create new game
        game_id = data.get("game_id", str(uuid.uuid4().int)[:10])
        queue_type_id = data.get("queue_type_id", player.queue_type_id)
        champion_id = data.get("champion_id", 1)
        
        game = MockGameInfo(
            game_id=game_id,
            game_queue_config_id=queue_type_id,
            participants=[{
                "puuid": player.puuid,
                "summonerName": player.game_name,
                "championId": champion_id,
                "teamId": 100
            }]
        )
        
        self.games[game_id] = game
        player.state = PlayerGameState.IN_GAME
        player.current_game_id = game_id
        player.current_game_start_time = game.game_start_time
        player.current_champion_id = champion_id
        
        logger.info("Started game for player", puuid=puuid, game_id=game_id)
        
        return web.json_response({
            "game_id": game_id,
            "state": player.state.value,
            "game_start_time": game.game_start_time
        })
        
    async def end_game(self, request: web.Request) -> web.Response:
        """End a game for a player with a result."""
        puuid = request.match_info['puuid']
        data = await request.json()
        
        if puuid not in self.players:
            return web.json_response({"error": "Player not found"}, status=404)
            
        player = self.players[puuid]
        
        if not player.current_game_id:
            return web.json_response({"error": "Player not in game"}, status=400)
            
        # Create match result
        won = data.get("won", True)
        duration_seconds = data.get("duration_seconds", 1800)  # Default 30 min
        kills = data.get("kills", 5)
        deaths = data.get("deaths", 3)
        assists = data.get("assists", 10)
        
        game = self.games.get(player.current_game_id)
        if not game:
            return web.json_response({"error": "Game not found"}, status=404)
            
        match_id = f"NA1_{player.current_game_id}"
        
        match_result = MockMatchResult(
            match_id=match_id,
            game_creation=game.game_start_time or int(time.time() * 1000),
            game_duration=duration_seconds,
            game_end_timestamp=int(time.time() * 1000),
            queue_id=game.game_queue_config_id,
            participants=[{
                "puuid": player.puuid,
                "riotIdGameName": player.game_name,
                "riotIdTagline": player.tag_line,
                "championName": f"Champion{player.current_champion_id}",
                "championId": player.current_champion_id,
                "win": won,
                "kills": kills,
                "deaths": deaths,
                "assists": assists
            }]
        )
        
        self.match_results[match_id] = match_result
        
        # Update player state
        player.state = PlayerGameState.NOT_IN_GAME
        player.last_match_result = {
            "match_id": match_id,
            "won": won,
            "duration_seconds": duration_seconds
        }
        
        # Clean up game
        del self.games[player.current_game_id]
        player.current_game_id = None
        player.current_game_start_time = None
        player.current_champion_id = None
        
        logger.info("Ended game for player", 
                   puuid=puuid, 
                   match_id=match_id,
                   won=won)
        
        return web.json_response({
            "match_id": match_id,
            "won": won,
            "state": player.state.value
        })
    
    async def get_current_tft_game(self, request: web.Request) -> web.Response:
        """Mock /lol/spectator/tft/v5/active-games/by-puuid endpoint."""
        await self.apply_request_delay()
        
        if rate_limit_response := await self.check_rate_limit():
            return rate_limit_response
            
        puuid = request.match_info['puuid']
        
        if puuid not in self.players:
            return web.json_response(
                {"status": {"message": "Player not found", "status_code": 404}},
                status=404
            )
            
        player = self.players[puuid]
        
        if player.state == PlayerGameState.NOT_IN_GAME:
            return web.json_response(
                {"status": {"message": "Player is not currently in a game", "status_code": 404}},
                status=404
            )
            
        # Return current TFT game info
        if player.current_game_id and player.current_game_id in self.games:
            game = self.games[player.current_game_id]
            # Update game length based on current time
            if game.game_start_time:
                game.game_length = int((time.time() * 1000 - game.game_start_time) / 1000)
            return web.json_response(game.to_api_response())
            
        # Create a default TFT game if none exists
        game_id = str(uuid.uuid4().int)[:10]
        game = MockGameInfo(
            game_id=game_id,
            game_queue_config_id=player.queue_type_id,
            participants=[{
                "puuid": player.puuid,
                "summonerName": player.game_name,
                "teamId": 100
            }]
        )
        self.games[game_id] = game
        player.current_game_id = game_id
        player.current_game_start_time = game.game_start_time
        
        return web.json_response(game.to_api_response())
    
    async def get_tft_match_info(self, request: web.Request) -> web.Response:
        """Mock /tft/match/v1/matches endpoint."""
        await self.apply_request_delay()
        
        if rate_limit_response := await self.check_rate_limit():
            return rate_limit_response
            
        match_id = request.match_info['match_id']
        
        if match_id in self.match_results:
            match = self.match_results[match_id]
            # Return TFT-specific match format
            tft_response = {
                "metadata": {
                    "match_id": match.match_id,
                    "participants": [p["puuid"] for p in match.participants]
                },
                "info": {
                    "game_datetime": match.game_creation,
                    "game_length": match.game_duration,
                    "game_variation": None,
                    "game_version": "Version 14.1",
                    "queue_id": match.queue_id,
                    "tft_game_type": "standard",
                    "tft_set_core_name": "TFTSet11",
                    "tft_set_name": "TFTSet11",
                    "tft_set_number": 11,
                    "participants": match.participants
                }
            }
            return web.json_response(tft_response)
            
        return web.json_response(
            {"status": {"message": "Match not found", "status_code": 404}},
            status=404
        )
    
    async def start_tft_game(self, request: web.Request) -> web.Response:
        """Start a TFT game for a player."""
        puuid = request.match_info['puuid']
        data = await request.json()
        
        if puuid not in self.players:
            return web.json_response({"error": "Player not found"}, status=404)
            
        player = self.players[puuid]
        
        # Create new TFT game
        game_id = data.get("game_id", str(uuid.uuid4().int)[:10])
        queue_type_id = data.get("queue_type_id", 1100)  # Default to ranked TFT
        
        game = MockGameInfo(
            game_id=game_id,
            game_queue_config_id=queue_type_id,
            participants=[{
                "puuid": player.puuid,
                "summonerName": player.game_name,
                "teamId": 100
            }]
        )
        
        self.games[game_id] = game
        player.state = PlayerGameState.IN_GAME
        player.current_game_id = game_id
        player.current_game_start_time = game.game_start_time
        
        logger.info("Started TFT game for player", puuid=puuid, game_id=game_id)
        
        return web.json_response({
            "game_id": game_id,
            "state": player.state.value,
            "game_start_time": game.game_start_time
        })
    
    async def end_tft_game(self, request: web.Request) -> web.Response:
        """End a TFT game for a player with a result."""
        puuid = request.match_info['puuid']
        data = await request.json()
        
        if puuid not in self.players:
            return web.json_response({"error": "Player not found"}, status=404)
            
        player = self.players[puuid]
        
        if not player.current_game_id:
            return web.json_response({"error": "Player not in game"}, status=400)
            
        # Create TFT match result
        placement = data.get("placement", 4)  # Default to 4th place
        duration_seconds = data.get("duration_seconds", 1800)  # Default 30 min
        
        game = self.games.get(player.current_game_id)
        if not game:
            return web.json_response({"error": "Game not found"}, status=404)
            
        match_id = f"NA1_{player.current_game_id}"
        
        # TFT-specific participant data
        match_result = MockMatchResult(
            match_id=match_id,
            game_creation=game.game_start_time or int(time.time() * 1000),
            game_duration=duration_seconds,
            game_end_timestamp=int(time.time() * 1000),
            queue_id=game.game_queue_config_id,
            participants=[{
                "puuid": player.puuid,
                "riotIdGameName": player.game_name,
                "riotIdTagline": player.tag_line,
                "placement": placement,
                "time_eliminated": duration_seconds if placement > 1 else 0
            }]
        )
        
        self.match_results[match_id] = match_result
        
        # Update player state
        player.state = PlayerGameState.NOT_IN_GAME
        player.last_match_result = {
            "match_id": match_id,
            "placement": placement,
            "duration_seconds": duration_seconds
        }
        
        # Clean up game
        del self.games[player.current_game_id]
        player.current_game_id = None
        player.current_game_start_time = None
        
        logger.info("Ended TFT game for player", 
                   puuid=puuid, 
                   match_id=match_id,
                   placement=placement)
        
        return web.json_response({
            "match_id": match_id,
            "placement": placement,
            "state": player.state.value
        })
        
    async def list_players(self, request: web.Request) -> web.Response:
        """List all mock players."""
        players_data = []
        for player in self.players.values():
            players_data.append({
                "puuid": player.puuid,
                "game_name": player.game_name,
                "tag_line": player.tag_line,
                "state": player.state.value,
                "current_game_id": player.current_game_id,
                "last_match_result": player.last_match_result
            })
            
        return web.json_response({"players": players_data})
        
    async def delete_player(self, request: web.Request) -> web.Response:
        """Delete a mock player."""
        puuid = request.match_info['puuid']
        
        if puuid not in self.players:
            return web.json_response({"error": "Player not found"}, status=404)
            
        player = self.players[puuid]
        
        # Clean up any active game
        if player.current_game_id and player.current_game_id in self.games:
            del self.games[player.current_game_id]
            
        del self.players[puuid]
        
        logger.info("Deleted mock player", puuid=puuid)
        
        return web.json_response({"status": "deleted"})
        
    async def update_settings(self, request: web.Request) -> web.Response:
        """Update server settings."""
        data = await request.json()
        
        if "request_delay" in data:
            self.request_delay = float(data["request_delay"])
            
        if "should_return_429" in data:
            self.should_return_429 = bool(data["should_return_429"])
            
        logger.info("Updated server settings", 
                   request_delay=self.request_delay,
                   should_return_429=self.should_return_429)
        
        return web.json_response({
            "request_delay": self.request_delay,
            "should_return_429": self.should_return_429
        })
        
    async def reset_server(self, request: web.Request) -> web.Response:
        """Reset server to initial state."""
        self.players.clear()
        self.games.clear()
        self.match_results.clear()
        self.request_delay = 0
        self.should_return_429 = False
        
        logger.info("Reset mock server to initial state")
        
        return web.json_response({"status": "reset"})
        
    def run(self):
        """Run the mock server."""
        logger.info("Starting mock Riot API server", port=self.port)
        web.run_app(self.app, host='0.0.0.0', port=self.port)


def main():
    """Main entry point."""
    import argparse
    
    parser = argparse.ArgumentParser(description='Mock Riot API Server')
    parser.add_argument('--port', type=int, default=8080, help='Port to run on')
    args = parser.parse_args()
    
    server = MockRiotAPIServer(port=args.port)
    server.run()


if __name__ == '__main__':
    main()