"""Factory for creating Riot API clients (real or mock).

This module provides a factory pattern for creating Riot API clients,
allowing easy switching between real and mock implementations based on
environment configuration.
"""

import os

import structlog

from .riot_api_client import RiotAPIClient
from ..config import Config

logger = structlog.get_logger()


class MockRiotAPIClient(RiotAPIClient):
    """Mock Riot API client that redirects to the mock server."""
    
    def __init__(self, api_key: str, mock_server_url: str, request_timeout: float = 10.0):
        """Initialize the mock Riot API client.
        
        Args:
            api_key: Riot API key (ignored for mock, but kept for compatibility)
            mock_server_url: URL of the mock server (e.g., http://localhost:8080)
            request_timeout: Request timeout in seconds
        """
        super().__init__(api_key, request_timeout)
        self.mock_server_url = mock_server_url.rstrip('/')
        logger.info("Using mock Riot API client", mock_server_url=self.mock_server_url)
        
    def _get_base_url(self, region: str) -> str:
        """Get the base URL for a region - returns mock server URL."""
        # For mock, we ignore region and always use the mock server
        return self.mock_server_url
        
    async def get_account_by_riot_id(self, game_name: str, tag_line: str):
        """Get account information by Riot ID from mock server."""
        # Mock server uses the same path structure
        url = f"{self.mock_server_url}/riot/account/v1/accounts/by-riot-id/{game_name}/{tag_line}"
        
        logger.info(
            "Fetching account from mock server",
            game_name=game_name,
            tag_line=tag_line,
            url=url
        )
        
        try:
            data = await self._make_request(url, handle_404_as="account_not_found")
            
            from .riot_api_client import SummonerInfo
            summoner_info = SummonerInfo(
                puuid=data["puuid"],
                game_name=data["gameName"],
                tag_line=data["tagLine"],
            )
            
            logger.info(
                "Successfully fetched account from mock",
                game_name=summoner_info.game_name,
                tag_line=summoner_info.tag_line,
                puuid=summoner_info.puuid,
            )
            
            return summoner_info
            
        except Exception as e:
            logger.error(
                "Error fetching account from mock",
                game_name=game_name,
                tag_line=tag_line,
                error=str(e),
            )
            raise
            
    def _get_regional_url(self, region: str) -> str:
        """Get the regional URL for Match API calls - returns mock server URL."""
        # For mock, we ignore region and always use the mock server
        return self.mock_server_url


def create_riot_api_client(config: Config) -> RiotAPIClient:
    """Factory function to create the appropriate Riot API client.
    
    Args:
        config: Application configuration
        
    Returns:
        RiotAPIClient instance (real or mock based on configuration)
    """
    # Check for mock server URL in environment or config
    mock_server_url = os.environ.get('MOCK_RIOT_API_URL')
    
    if mock_server_url:
        logger.info("Creating mock Riot API client", mock_server_url=mock_server_url)
        return MockRiotAPIClient(
            api_key=config.riot_api_key,
            mock_server_url=mock_server_url,
            request_timeout=config.riot_api_timeout_seconds
        )
    else:
        logger.info("Creating real Riot API client")
        return RiotAPIClient(
            api_key=config.riot_api_key,
            request_timeout=config.riot_api_timeout_seconds
        )


def is_using_mock_api() -> bool:
    """Check if the mock API is being used."""
    return bool(os.environ.get('MOCK_RIOT_API_URL'))