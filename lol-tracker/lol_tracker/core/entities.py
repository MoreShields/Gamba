"""Core entities for the lol-tracker service."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional

from .enums import GameStatus, QueueType


@dataclass
class Player:
    """Represents a tracked League of Legends player.
    
    Simplified entity that combines all player tracking information
    without unnecessary abstraction layers.
    """
    
    # Riot ID components (replaces SummonerIdentity value object)
    game_name: str
    tag_line: str
    puuid: Optional[str] = None
    
    # Tracking metadata
    created_at: datetime = field(default_factory=datetime.utcnow)
    updated_at: datetime = field(default_factory=datetime.utcnow)
    
    # Database ID
    id: Optional[int] = None
    
    @property
    def riot_id(self) -> str:
        """Get the player's Riot ID in game_name#tag_line format."""
        return f"{self.game_name}#{self.tag_line}"
    
    def can_be_tracked(self) -> bool:
        """Check if this player can be actively tracked."""
        return (
            self.puuid is not None
            and bool(self.game_name.strip())
            and bool(self.tag_line.strip())
        )
    
    
    def __str__(self) -> str:
        """String representation of the player."""
        return f"Player({self.riot_id})"


@dataclass
class GameState:
    """Represents a player's current game state.
    
    Simplified entity that tracks game status and results without
    unnecessary complexity around state transitions.
    """
    
    # Core state information
    status: GameStatus
    player_id: int
    
    # Game information
    game_id: Optional[str] = None
    queue_type: Optional[QueueType] = None
    
    # Game result information
    won: Optional[bool] = None
    duration_seconds: Optional[int] = None
    champion_played: Optional[str] = None
    
    # Timestamps
    created_at: datetime = field(default_factory=datetime.utcnow)
    game_start_time: Optional[datetime] = None
    game_end_time: Optional[datetime] = None
    
    # Database ID
    id: Optional[int] = None
    
    @classmethod
    def create_not_in_game(cls, player_id: int) -> "GameState":
        """Create a not-in-game state."""
        return cls(
            status=GameStatus.NOT_IN_GAME,
            player_id=player_id
        )
    
    @classmethod
    def create_in_game(
        cls,
        player_id: int,
        game_id: str,
        queue_type: QueueType,
        game_start_time: Optional[datetime] = None
    ) -> "GameState":
        """Create an in-game state."""
        return cls(
            status=GameStatus.IN_GAME,
            player_id=player_id,
            game_id=game_id,
            queue_type=queue_type,
            game_start_time=game_start_time or datetime.utcnow()
        )
    
    def update_game_result(
        self,
        won: bool,
        duration_seconds: int,
        champion_played: str
    ) -> None:
        """Update game result information."""
        if self.status != GameStatus.NOT_IN_GAME:
            raise ValueError("Cannot update result for game still in progress")
        
        self.won = won
        self.duration_seconds = duration_seconds
        self.champion_played = champion_played
    
    @property
    def is_active_game(self) -> bool:
        """Check if this represents an active game."""
        return self.status.is_playing
    
    def __str__(self) -> str:
        """String representation of the game state."""
        game_info = f"game_id={self.game_id}" if self.game_id else "no_game"
        return f"GameState({self.status.value}, {game_info})"