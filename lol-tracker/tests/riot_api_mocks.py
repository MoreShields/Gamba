"""Mock utilities for Riot API responses in tests."""

import time
from typing import Dict, Any, Optional
from unittest.mock import Mock

import httpx
import respx


class RiotAPIMockData:
    """Collection of mock data for Riot API responses."""

    @staticmethod
    def get_account_response(
        game_name: str = "TestSummoner",
        tag_line: str = "NA1",
        puuid: str = "test_puuid_123",
    ) -> Dict[str, Any]:
        """Generate a mock account response from Riot API."""
        return {
            "puuid": puuid,
            "gameName": game_name,
            "tagLine": tag_line,
        }

    @staticmethod
    def get_error_response(status_code: int, message: str = "Error") -> Dict[str, Any]:
        """Generate a mock error response."""
        return {"status": {"message": message, "status_code": status_code}}


class RiotAPIMockRouter:
    """Mock router for Riot API endpoints using respx."""

    def __init__(self):
        self.router = respx.Router(base_url="https://NA1.api.riotgames.com")
        self.mock_responses = {}

    def mock_get_account_by_riot_id(
        self,
        game_name: str,
        tag_line: str,
        status_code: int = 200,
        response_data: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None,
    ):
        """Mock a get account by riot id endpoint."""
        if response_data is None:
            response_data = RiotAPIMockData.get_account_response(
                game_name=game_name,
                tag_line=tag_line
            )

        # Account API uses regional routing - determine region from game_name context if needed
        # For now, default to americas since we don't have region parameter
        regional = "americas"
        url_pattern = f"https://{regional}.api.riotgames.com/riot/account/v1/accounts/by-riot-id/{game_name}/{tag_line}"

        mock_headers = {"Content-Type": "application/json"}
        if headers:
            mock_headers.update(headers)

        self.router.get(url_pattern).mock(
            return_value=httpx.Response(
                status_code=status_code, json=response_data, headers=mock_headers
            )
        )

    def mock_account_not_found(self, game_name: str, tag_line: str):
        """Mock an account not found response (404)."""
        self.mock_get_account_by_riot_id(
            game_name=game_name,
            tag_line=tag_line,
            status_code=404,
            response_data=RiotAPIMockData.get_error_response(404, "Account not found"),
        )

    def mock_rate_limit_error(
        self, game_name: str, tag_line: str, retry_after: int = 60
    ):
        """Mock a rate limit error response (429)."""
        self.mock_get_account_by_riot_id(
            game_name=game_name,
            tag_line=tag_line,
            status_code=429,
            response_data=RiotAPIMockData.get_error_response(
                429, "Rate limit exceeded"
            ),
            headers={"Retry-After": str(retry_after)},
        )

    def mock_api_error(
        self, game_name: str, tag_line: str, status_code: int = 500
    ):
        """Mock a general API error response."""
        self.mock_get_account_by_riot_id(
            game_name=game_name,
            tag_line=tag_line,
            status_code=status_code,
            response_data=RiotAPIMockData.get_error_response(
                status_code, "Internal server error"
            ),
        )

    def start(self):
        """Start the mock router."""
        return self.router.start()

    def stop(self):
        """Stop the mock router."""
        return self.router.stop()

    def __enter__(self):
        """Context manager entry."""
        self.start()
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.stop()


# Convenience functions for common mock scenarios
def mock_successful_summoner_lookup(
    game_name: str = "TestSummoner", tag_line: str = "NA1", **summoner_data
):
    """Create a mock for successful summoner lookup."""
    router = RiotAPIMockRouter()

    response_data = RiotAPIMockData.get_account_response(
        game_name=game_name, tag_line=tag_line, **summoner_data
    )
    router.mock_get_account_by_riot_id(
        game_name=game_name, tag_line=tag_line, response_data=response_data
    )

    return router


def mock_summoner_not_found(
    game_name: str = "NonExistentSummoner", tag_line: str = "NA1"
):
    """Create a mock for summoner not found."""
    router = RiotAPIMockRouter()
    router.mock_account_not_found(game_name, tag_line)
    return router


def mock_rate_limit_error(
    game_name: str = "TestSummoner", tag_line: str = "NA1", retry_after: int = 60
):
    """Create a mock for rate limit error."""
    router = RiotAPIMockRouter()
    router.mock_rate_limit_error(game_name, tag_line, retry_after)
    return router


def mock_api_error(
    game_name: str = "TestSummoner", tag_line: str = "NA1", status_code: int = 500
):
    """Create a mock for API error."""
    router = RiotAPIMockRouter()
    router.mock_api_error(game_name, tag_line, status_code)
    return router


# Legacy mock support using Mock objects (for compatibility with existing tests)
def create_mock_riot_api_client():
    """Create a Mock RiotAPIClient for unit testing."""
    mock_client = Mock()

    # Configure default successful response
    mock_client.get_summoner_by_name.return_value = Mock(
        puuid="test_puuid_123",
        game_name="TestSummoner", 
        tag_line="NA1"
    )

    return mock_client
