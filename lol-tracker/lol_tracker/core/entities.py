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
    
    @staticmethod
    def parse_game_result(queue_type: Optional[QueueType], result_data: Optional[dict]) -> Optional[GameResult]:
        """Parse game result from JSON data based on queue type.
        
        Args:
            queue_type: The queue type to determine game type
            result_data: Raw JSON data from database
            
        Returns:
            Appropriate GameResult object or None
        """
        if not result_data or not queue_type:
            return None
            
        if queue_type.game_type == GameType.LOL:
            return LoLGameResult(
                won=result_data.get('won', False),
                duration_seconds=result_data.get('duration_seconds', 0),
                champion_played=result_data.get('champion_played', '')
            )
        elif queue_type.game_type == GameType.TFT:
            return TFTGameResult(
                placement=result_data.get('placement', 8),
                duration_seconds=result_data.get('duration_seconds', 0)
            )
        
        return None
    
    def update_game_result(self, game_result: GameResult) -> None:
        """Update game result information."""
        if self.status != GameStatus.NOT_IN_GAME:
            raise ValueError("Cannot update result for game still in progress")
        
        self.game_result = game_result
    
    def create_state_change_event(self, player: "Player", previous_state: "GameState") -> GameStateChangedEvent:
        """Create appropriate event for this state change based on game type.
        
        This method encapsulates all game-type-specific logic for event creation,
        allowing the application layer to remain completely agnostic.
        
        Args:
            player: The player whose state changed
            previous_state: The previous game state
            
        Returns:
            Appropriate GameStateChangedEvent subclass based on game type
        """
        is_game_start = (previous_state.status == GameStatus.NOT_IN_GAME 
                         and self.status == GameStatus.IN_GAME)
        is_game_end = (previous_state.status == GameStatus.IN_GAME 
                       and self.status == GameStatus.NOT_IN_GAME)
        
        # Common event data for all game types
        common_kwargs = {
            'player_id': player.id,
            'game_name': player.game_name,
            'tag_line': player.tag_line,
            'puuid': player.puuid,
            'previous_status': previous_state.status.value,
            'new_status': self.status.value,
            'game_id': self.game_id or previous_state.game_id,
            'queue_type': self.queue_type.value if self.queue_type else None,
            'changed_at': datetime.utcnow(),
            'is_game_start': is_game_start,
            'is_game_end': is_game_end,
            'duration_seconds': self.game_result.duration_seconds if self.game_result else None
        }
        
        # Create appropriate event type based on game type
        if self.queue_type and self.queue_type.game_type == GameType.LOL:
            return LoLGameStateChangedEvent(
                **common_kwargs,
                won=self.game_result.won if isinstance(self.game_result, LoLGameResult) else None,
                champion_played=self.game_result.champion_played if isinstance(self.game_result, LoLGameResult) else None
            )
        elif self.queue_type and self.queue_type.game_type == GameType.TFT:
            return TFTGameStateChangedEvent(
                **common_kwargs,
                placement=self.game_result.placement if isinstance(self.game_result, TFTGameResult) else None
            )
        else:
            # Default to LoL event for unknown queue types
            return LoLGameStateChangedEvent(**common_kwargs)
    
    def update_from_match_info(self, match_info: Any, puuid: str) -> Optional[GameResult]:
        """Update game state from match information polymorphically.
        
        This method encapsulates all game-type-specific logic for processing
        match results, allowing the application layer to remain agnostic.
        
        Args:
            match_info: Match information object (MatchInfo or TFTMatchInfo)
            puuid: Player's PUUID to find in participants
            
        Returns:
            GameResult if successfully updated, None if player not found
            
        Raises:
            ValueError: If game state is not ready for results
        """
        if self.status != GameStatus.NOT_IN_GAME:
            raise ValueError("Cannot update result for game still in progress")
        
        if not self.queue_type:
            raise ValueError("Cannot determine game type without queue type")
        
        game_type = self.queue_type.game_type
        
        # Build appropriate game result based on game type
        if game_type == GameType.LOL:
            # Handle LoL match
            if hasattr(match_info, 'get_participant_result'):
                participant_result = match_info.get_participant_result(puuid)
                if not participant_result:
                    return None
                
                game_result = LoLGameResult(
                    won=participant_result["won"],
                    duration_seconds=match_info.game_duration,
                    champion_played=participant_result["champion_name"]
                )
            else:
                return None
                
        elif game_type == GameType.TFT:
            # Handle TFT match
            if hasattr(match_info, 'get_placement'):
                placement = match_info.get_placement(puuid)
                if placement is None:
                    return None
                
                game_result = TFTGameResult(
                    placement=placement,
                    duration_seconds=int(match_info.game_length)
                )
            else:
                return None
        else:
            return None
        
        # Update self with the result
        self.update_game_result(game_result)
        return game_result
    
    
    @property
    def game_type(self) -> Optional[GameType]:
        """Get game type from queue type's metadata.
        
        Returns None when there's no queue type (e.g., when player is not in game).
        """
        if self.queue_type:
            return self.queue_type.game_type
        return None  # No game type when not in a game
    
    @property
    def is_active_game(self) -> bool:
        """Check if this represents an active game."""
        return self.status.is_playing
    
    def __str__(self) -> str:
        """String representation of the game state."""
        game_info = f"game_id={self.game_id}" if self.game_id else "no_game"
        return f"GameState({self.status.value}, {game_info})"