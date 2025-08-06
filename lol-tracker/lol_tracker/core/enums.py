"""Core enums for the lol-tracker service."""

from enum import Enum
from typing import Dict, Optional, Set, Tuple


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
    """League of Legends and TFT queue types with associated metadata.
    
    Each queue type contains:
    - value: String identifier for the queue
    - game_type: The GameType this queue belongs to
    - queue_id: The Riot API queue ID
    
    Data from: https://raw.communitydragon.org/latest/plugins/rcp-be-lol-game-data/global/default/v1/queues.json
    """
    
    # Format: (string_value, game_type, queue_id)
    
    # Core Summoner's Rift PvP
    NORMAL_DRAFT = ("NORMAL_DRAFT", GameType.LOL, 400)           # Normal (Draft Pick)
    RANKED_SOLO_5X5 = ("RANKED_SOLO_5x5", GameType.LOL, 420)    # Ranked Solo/Duo
    NORMAL_BLIND = ("NORMAL_BLIND", GameType.LOL, 430)           # Normal (Blind Pick)
    RANKED_FLEX_SR = ("RANKED_FLEX_SR", GameType.LOL, 440)      # Ranked Flex
    QUICKPLAY = ("QUICKPLAY", GameType.LOL, 490)                 # Quickplay
    
    # Alternative LoL modes
    ARAM = ("ARAM", GameType.LOL, 450)                           # ARAM
    CLASH = ("CLASH", GameType.LOL, 700)                         # Clash tournament
    ARENA = ("ARENA", GameType.LOL, 1700)                        # Arena 2v2v2v2
    
    # Rotating game modes
    URF = ("URF", GameType.LOL, 1900)                            # Ultra Rapid Fire
    ARURF = ("ARURF", GameType.LOL, 900)                         # All Random URF
    ONE_FOR_ALL = ("ONE_FOR_ALL", GameType.LOL, 1020)            # One For All
    ULTIMATE_SPELLBOOK = ("ULTIMATE_SPELLBOOK", GameType.LOL, 1400)  # Ultimate Spellbook
    NEXUS_BLITZ = ("NEXUS_BLITZ", GameType.LOL, 1300)           # Nexus Blitz
    
    # Bot games
    BOT_INTRO_5X5 = ("BOT_INTRO_5x5", GameType.LOL, 830)        # Intro bots
    BOT_BEGINNER_5X5 = ("BOT_BEGINNER_5x5", GameType.LOL, 840)  # Beginner bots
    BOT_INTERMEDIATE_5X5 = ("BOT_INTERMEDIATE_5x5", GameType.LOL, 850)  # Intermediate bots
    
    # TFT queue types
    TFT_NORMAL = ("TFT_NORMAL", GameType.TFT, 1090)              # Teamfight Tactics (Normal)
    TFT_RANKED = ("TFT_RANKED", GameType.TFT, 1100)              # Teamfight Tactics (Ranked)
    TFT_TUTORIAL = ("TFT_TUTORIAL", GameType.TFT, 1110)          # Teamfight Tactics (Tutorial)
    TFT_HYPER_ROLL = ("TFT_HYPER_ROLL", GameType.TFT, 1130)      # Teamfight Tactics (Hyper Roll)
    TFT_DOUBLE_UP = ("TFT_DOUBLE_UP", GameType.TFT, 1150)        # Teamfight Tactics (Double Up workshop)
    TFT_NORMAL_HYPER_ROLL = ("TFT_NORMAL_HYPER_ROLL", GameType.TFT, 1120)  # Teamfight Tactics (Normal Hyper Roll)
    TFT_NORMAL_DOUBLE_UP = ("TFT_NORMAL_DOUBLE_UP", GameType.TFT, 1140)    # Teamfight Tactics (Normal Double Up)
    
    def __init__(self, value: str, game_type: GameType, queue_id: int):
        """Initialize queue type with metadata."""
        self._value_ = value
        self.game_type = game_type
        self.queue_id = queue_id
    
    @property
    def value(self) -> str:
        """Get the string value of the queue type."""
        return self._value_
    
    @classmethod
    def from_queue_id(cls, queue_id: int) -> Optional["QueueType"]:
        """Convert a Riot API queue ID to a QueueType.
        
        Returns None for unknown queue IDs.
        """
        for queue_type in cls:
            if queue_type.queue_id == queue_id:
                return queue_type
        return None
    
    @classmethod
    def from_string(cls, value: str) -> Optional["QueueType"]:
        """Convert a string value to a QueueType.
        
        Returns None for unknown values.
        """
        for queue_type in cls:
            if queue_type.value == value:
                return queue_type
        return None
    
