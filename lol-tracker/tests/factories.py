"""Test data factories for creating test objects."""
from datetime import datetime
from typing import Optional

from lol_tracker.database.models import TrackedPlayer, GameState


class TrackedPlayerFactory:
    """Factory for creating TrackedPlayer test instances."""
    
    @staticmethod
    def create(
        summoner_name: str = "TestSummoner",
        region: str = "NA1",
        puuid: Optional[str] = None,
        account_id: Optional[str] = None,
        summoner_id: Optional[str] = None,
        is_active: bool = True,
    ) -> TrackedPlayer:
        """Create a TrackedPlayer instance with test data."""
        return TrackedPlayer(
            summoner_name=summoner_name,
            region=region,
            puuid=puuid or f"test_puuid_{summoner_name}",
            account_id=account_id or f"test_account_{summoner_name}",
            summoner_id=summoner_id or f"test_summoner_{summoner_name}",
            is_active=is_active,
        )
    
    @staticmethod
    def create_multiple(count: int, base_name: str = "Player") -> list[TrackedPlayer]:
        """Create multiple TrackedPlayer instances."""
        return [
            TrackedPlayerFactory.create(
                summoner_name=f"{base_name}{i}",
                puuid=f"test_puuid_{i}",
                account_id=f"test_account_{i}",
                summoner_id=f"test_summoner_{i}",
            )
            for i in range(1, count + 1)
        ]


class GameStateFactory:
    """Factory for creating GameState test instances."""
    
    @staticmethod
    def create(
        player_id: int,
        status: str = "NOT_IN_GAME",
        game_id: Optional[str] = None,
        queue_type: Optional[str] = None,
        won: Optional[bool] = None,
        duration_seconds: Optional[int] = None,
        champion_played: Optional[str] = None,
        game_start_time: Optional[datetime] = None,
        game_end_time: Optional[datetime] = None,
        raw_api_response: Optional[str] = None,
    ) -> GameState:
        """Create a GameState instance with test data."""
        return GameState(
            player_id=player_id,
            status=status,
            game_id=game_id,
            queue_type=queue_type,
            won=won,
            duration_seconds=duration_seconds,
            champion_played=champion_played,
            game_start_time=game_start_time,
            game_end_time=game_end_time,
            raw_api_response=raw_api_response,
        )
    
    @staticmethod
    def create_game_sequence(player_id: int) -> list[GameState]:
        """Create a sequence of game states representing a typical game flow."""
        return [
            GameStateFactory.create(
                player_id=player_id,
                status="IN_CHAMPION_SELECT",
                game_id="test_game_123",
                queue_type="RANKED_SOLO_5x5",
            ),
            GameStateFactory.create(
                player_id=player_id,
                status="IN_GAME",
                game_id="test_game_123",
                queue_type="RANKED_SOLO_5x5",
                game_start_time=datetime.utcnow(),
            ),
            GameStateFactory.create(
                player_id=player_id,
                status="NOT_IN_GAME",
                game_id="test_game_123",
                queue_type="RANKED_SOLO_5x5",
                won=True,
                duration_seconds=1800,
                champion_played="Jinx",
                game_end_time=datetime.utcnow(),
            ),
        ]