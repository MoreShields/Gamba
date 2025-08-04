"""Riot API adapter package.

This package contains the Riot API client adapter for the LoL Tracker service.
"""

from .client import (
    RiotAPIClient,
    RiotAPIError,
    SummonerNotFoundError,
    RateLimitError,
    InvalidRegionError,
    PlayerNotInGameError,
    SummonerInfo,
    CurrentGameInfo,
    MatchInfo,
    RiotRegion,
)

__all__ = [
    # Client
    "RiotAPIClient",
    "RiotRegion",
    # Exceptions
    "RiotAPIError",
    "SummonerNotFoundError",
    "RateLimitError",
    "InvalidRegionError",
    "PlayerNotInGameError",
    # Data classes
    "SummonerInfo",
    "CurrentGameInfo",
    "MatchInfo",
]