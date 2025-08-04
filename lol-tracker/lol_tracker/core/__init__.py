"""Simplified core layer for the lol-tracker service.

This module provides a pragmatic alternative to the complex clean architecture
implementation, focusing on essential functionality without unnecessary abstraction.
"""

from .entities import Player, GameState
from .enums import GameStatus, QueueType
from .services import GameStateTransitionService

__all__ = [
    "Player",
    "GameState", 
    "GameStatus",
    "QueueType",
    "GameStateTransitionService",
]