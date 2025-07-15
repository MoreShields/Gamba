"""Protobuf imports for LoL Tracker service."""

# Import generated protobuf modules
from .events import lol_events_pb2, tracking_events_pb2
from .models import common_pb2

# Re-export key classes for easier imports
from .events.lol_events_pb2 import (
    GameStatus,
    LoLGameStateChanged,
    GameResult,
)

from .events.tracking_events_pb2 import (
    PlayerTrackingCommand,
    StartTrackingPlayer,
    StopTrackingPlayer,
)

__all__ = [
    # Modules
    "lol_events_pb2",
    "tracking_events_pb2", 
    "common_pb2",
    
    # Enums
    "GameStatus",
    
    # Messages
    "LoLGameStateChanged",
    "GameResult",
    "PlayerTrackingCommand",
    "StartTrackingPlayer",
    "StopTrackingPlayer",
]