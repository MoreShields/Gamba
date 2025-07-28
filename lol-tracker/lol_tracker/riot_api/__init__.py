"""Riot API integration package.

This package contains all Riot API related functionality including:
- Real Riot API client
- Mock Riot API server for testing
- Factory for switching between real and mock implementations
"""

from .riot_api_client import (
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
from .riot_api_factory import create_riot_api_client, is_using_mock_api
from .mock_riot_server import MockRiotAPIServer
from .mock_riot_control import MockRiotControlClient, GameState

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
    # Factory
    "create_riot_api_client",
    "is_using_mock_api",
    # Mock
    "MockRiotAPIServer",
    "MockRiotControlClient",
    "GameState",
]