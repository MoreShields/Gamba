"""Core entities for the lol-tracker service."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional, Union

from .enums import GameStatus, QueueType, GameType


@dataclass
class LoLGameResult:
    """Represents the result of a completed League of Legends game."""
    
    won: bool
    duration_seconds: int
    champion_played: str


@dataclass 
class TFTGameResult:
    """Represents the result of a completed Teamfight Tactics game."""
    
    placement: int  # 1-8 placement
    duration_seconds: int
    little_legend: Optional[str] = None


# Union type for polymorphic game results
GameResult = Union[LoLGameResult, TFTGameResult]


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
    game_type: GameType = GameType.LOL
    
    # Game result information
    game_result: Optional[GameResult] = None
    
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
        game_start_time: Optional[datetime] = None,
        game_type: Optional[GameType] = None
    ) -> "GameState":
        """Create an in-game state."""
        # Auto-determine game type from queue if not explicitly provided
        if game_type is None:
            game_type = cls._determine_game_type_from_queue(queue_type)
            
        return cls(
            status=GameStatus.IN_GAME,
            player_id=player_id,
            game_id=game_id,
            queue_type=queue_type,
            game_type=game_type,
            game_start_time=game_start_time or datetime.utcnow()
        )
    
    def update_game_result(self, game_result: GameResult) -> None:
        """Update game result information."""
        if self.status != GameStatus.NOT_IN_GAME:
            raise ValueError("Cannot update result for game still in progress")
        
        self.game_result = game_result
    
    
    @classmethod
    def _determine_game_type_from_queue(cls, queue_type: Optional[QueueType]) -> GameType:
        """Determine game type from queue type."""
        if queue_type in [QueueType.RANKED_TFT, QueueType.RANKED_TFT_TURBO, QueueType.RANKED_TFT_DOUBLE_UP]:
            return GameType.TFT
        return GameType.LOL
    
    # Backward compatibility properties for existing code
    @property
    def won(self) -> Optional[bool]:
        """Get win status from game result (LoL only)."""
        if isinstance(self.game_result, LoLGameResult):
            return self.game_result.won
        elif isinstance(self.game_result, TFTGameResult):
            # TFT: top 4 is generally considered a "win"
            return self.game_result.placement <= 4
        return None
    
    @property
    def duration_seconds(self) -> Optional[int]:
        """Get game duration from game result."""
        if self.game_result:
            return self.game_result.duration_seconds
        return None
    
    @property 
    def champion_played(self) -> Optional[str]:
        """Get champion played from game result (LoL only)."""
        if isinstance(self.game_result, LoLGameResult):
            return self.game_result.champion_played
        return None
    
    @property
    def is_active_game(self) -> bool:
        """Check if this represents an active game."""
        return self.status.is_playing
    
    def __str__(self) -> str:
        """String representation of the game state."""
        game_info = f"game_id={self.game_id}" if self.game_id else "no_game"
        return f"GameState({self.status.value}, {game_info})"