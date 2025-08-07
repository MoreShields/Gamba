"""Domain events for game state changes."""

from abc import ABC, abstractmethod
from dataclasses import dataclass
from datetime import datetime
from typing import Optional


@dataclass
class GameStateChangedEvent(ABC):
    """Abstract base for all game state change events.
    
    This domain event represents a player's game state transition and
    encapsulates all the data needed for downstream consumers.
    """
    # Required fields first
    player_id: int
    game_name: str
    tag_line: str
    previous_status: str
    new_status: str
    is_game_start: bool
    is_game_end: bool
    changed_at: datetime
    
    # Optional fields
    game_id: Optional[str] = None
    queue_type: Optional[str] = None
    duration_seconds: Optional[int] = None
    
    @abstractmethod
    def get_event_type(self) -> str:
        """Get the event type identifier for routing."""
        pass


@dataclass
class LoLGameStateChangedEvent(GameStateChangedEvent):
    """League of Legends specific game state change event.
    
    Includes LoL-specific fields like champion played and win/loss.
    """
    won: Optional[bool] = None
    champion_played: Optional[str] = None
    
    def get_event_type(self) -> str:
        return "lol.game_state_changed"


@dataclass
class TFTGameStateChangedEvent(GameStateChangedEvent):
    """Teamfight Tactics specific game state change event.
    
    Includes TFT-specific fields like placement.
    """
    placement: Optional[int] = None
    
    def get_event_type(self) -> str:
        return "tft.game_state_changed"
    
    @property
    def won(self) -> Optional[bool]:
        """In TFT, top 4 placement is considered a win."""
        if self.placement is not None:
            return self.placement <= 4
        return None