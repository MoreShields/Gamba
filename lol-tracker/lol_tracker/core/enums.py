"""Core enums for the lol-tracker service."""

from enum import Enum
from typing import Dict, Set


class GameType(Enum):
    """Type of game being tracked."""
    
    LOL = "LOL"
    TFT = "TFT"


class GameStatus(Enum):
    """Player's current game status."""
    
    NOT_IN_GAME = "NOT_IN_GAME"
    IN_GAME = "IN_GAME"
    
    @property
    def is_playing(self) -> bool:
        """Check if the player is actively in a game."""
        return self == self.IN_GAME
    
    def can_transition_to(self, new_status: "GameStatus") -> bool:
        """Check if a transition to the new status is valid."""
        return new_status in self._valid_transitions()
    
    def _valid_transitions(self) -> Set["GameStatus"]:
        """Get valid status transitions from current status."""
        if self == self.NOT_IN_GAME:
            return {self.IN_GAME}
        elif self == self.IN_GAME:
            return {self.NOT_IN_GAME}
        return set()


class QueueType(Enum):
    """League of Legends and TFT queue types."""
    
    # LoL queue types
    RANKED_SOLO_5X5 = "RANKED_SOLO_5x5"
    RANKED_FLEX_SR = "RANKED_FLEX_SR"
    ARAM = "ARAM"
    NORMAL_DRAFT = "NORMAL_DRAFT"
    NORMAL_BLIND = "NORMAL_BLIND"
    CLASH = "CLASH"
    ARENA = "ARENA"
    
    # TFT queue types
    RANKED_TFT = "RANKED_TFT"
    RANKED_TFT_TURBO = "RANKED_TFT_TURBO"
    RANKED_TFT_DOUBLE_UP = "RANKED_TFT_DOUBLE_UP"
    
    UNKNOWN = "UNKNOWN"
    
    @classmethod
    def from_queue_id(cls, queue_id: int) -> "QueueType":
        """Convert a Riot API queue ID to a QueueType."""
        queue_map: Dict[int, QueueType] = {
            # LoL queue IDs
            420: cls.RANKED_SOLO_5X5,
            440: cls.RANKED_FLEX_SR,
            450: cls.ARAM,
            400: cls.NORMAL_DRAFT,
            430: cls.NORMAL_BLIND,
            700: cls.CLASH,
            1700: cls.ARENA,
            # TFT queue IDs
            1100: cls.RANKED_TFT,
            1130: cls.RANKED_TFT_TURBO,
            1150: cls.RANKED_TFT_DOUBLE_UP,
        }
        return queue_map.get(queue_id, cls.UNKNOWN)
    
