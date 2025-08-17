"""Core entities for the lol-tracker service."""

from dataclasses import dataclass, field
from datetime import datetime
from typing import Optional, Union, Any

from .enums import GameStatus, QueueType, GameType
from .events import GameStateChangedEvent, LoLGameStateChangedEvent, TFTGameStateChangedEvent


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
            bool(self.game_name.strip())
            and bool(self.tag_line.strip())
        )
    
    
    def __str__(self) -> str:
        """String representation of the player."""
        return f"Player({self.riot_id})"


@dataclass
class TrackedGame:
    """Represents a tracked game in the game-centric model.
    
    Each game is tracked as a single entity that progresses through
    its lifecycle from detection to completion.
    """
    
    # Core identification
    player_id: int
    game_id: str
    
    # Status - simplified to just ACTIVE/COMPLETED
    status: str  # 'ACTIVE' or 'COMPLETED'
    
    # Timestamps
    detected_at: datetime = field(default_factory=datetime.utcnow)
    started_at: Optional[datetime] = None
    completed_at: Optional[datetime] = None
    
    # Game information
    game_type: str = 'LOL'  # 'LOL' or 'TFT'
    queue_type: Optional[QueueType] = None
    game_result: Optional[GameResult] = None
    duration_seconds: Optional[int] = None
    
    # Metadata
    last_error: Optional[str] = None
    raw_api_response: Optional[str] = None
    
    # Database ID
    id: Optional[int] = None
    
    def is_active(self) -> bool:
        """Check if game is still active."""
        return self.status == 'ACTIVE'
    
    def is_completed(self) -> bool:
        """Check if game has completed."""
        return self.status == 'COMPLETED'
    
    def complete_with_results(
        self, 
        match_info: Any,
        game_name: str,
        tag_line: str
    ) -> bool:
        """Complete the game with match results.
        
        Args:
            match_info: MatchInfo or TFTMatchInfo from Riot API
            game_name: Player's game name
            tag_line: Player's tag line
            
        Returns:
            True if results were successfully processed, False otherwise
        """
        if not self.is_active():
            raise ValueError(f"Cannot complete game {self.game_id} - not active")
        
        # Extract result based on game type
        if self.game_type == 'LOL':
            # LoL game - check for get_participant_result_by_name method
            if hasattr(match_info, 'get_participant_result_by_name'):
                participant = match_info.get_participant_result_by_name(game_name, tag_line)
                if participant:
                    self.game_result = LoLGameResult(
                        won=participant["won"],
                        duration_seconds=match_info.game_duration,
                        champion_played=participant["champion_name"]
                    )
                    self.duration_seconds = match_info.game_duration
        elif self.game_type == 'TFT':
            # TFT game - check for get_placement_by_name method
            if hasattr(match_info, 'get_placement_by_name'):
                placement = match_info.get_placement_by_name(game_name, tag_line)
                if placement is not None:
                    self.game_result = TFTGameResult(
                        placement=placement,
                        duration_seconds=int(match_info.game_length)
                    )
                    self.duration_seconds = int(match_info.game_length)
        
        if self.game_result:
            self.status = 'COMPLETED'
            self.completed_at = datetime.utcnow()
            return True
        
        return False
    
    def mark_error(self, error: str) -> None:
        """Mark that an error occurred while processing this game."""
        self.last_error = error
    
    def __str__(self) -> str:
        """String representation of the tracked game."""
        return f"TrackedGame(game_id={self.game_id}, status={self.status}, player_id={self.player_id})"