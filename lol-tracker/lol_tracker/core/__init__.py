"""Simplified core layer for the lol-tracker service.

This module provides the core domain entities and enums for the game-centric
tracking model.
"""

from .entities import Player, TrackedGame, LoLGameResult, TFTGameResult
from .enums import GameStatus, QueueType, GameType

__all__ = [
    "Player",
    "TrackedGame",
    "LoLGameResult",
    "TFTGameResult",
    "GameStatus",
    "QueueType",
    "GameType",
]