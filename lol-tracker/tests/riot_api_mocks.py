"""Mock utilities for Riot API responses in tests."""

import time
from typing import Dict, Any, Optional
from unittest.mock import Mock

import httpx
import respx


class RiotAPIMockData:
    """Collection of mock data for Riot API responses."""
    
    @staticmethod
    def get_summoner_response(
        summoner_name: str = "TestSummoner",
        puuid: str = "test_puuid_123",
        account_id: str = "test_account_123",
        summoner_id: str = "test_summoner_123",
        summoner_level: int = 100,
        revision_date: Optional[int] = None
    ) -> Dict[str, Any]:
        """Generate a mock summoner response from Riot API."""
        if revision_date is None:
            revision_date = int(time.time() * 1000)
            
        return {
            "id": summoner_id,
            "accountId": account_id,
            "puuid": puuid,
            "name": summoner_name,
            "profileIconId": 1234,
            "revisionDate": revision_date,
            "summonerLevel": summoner_level
        }

    @staticmethod
    def get_error_response(status_code: int, message: str = "Error") -> Dict[str, Any]:
        """Generate a mock error response."""
        return {
            "status": {
                "message": message,
                "status_code": status_code
            }
        }


class RiotAPIMockRouter:
    """Mock router for Riot API endpoints using respx."""
    
    def __init__(self):
        self.router = respx.Router(base_url="https://na1.api.riotgames.com")
        self.mock_responses = {}
        
    def mock_get_summoner_by_name(
        self,
        summoner_name: str,
        region: str = "na1",
        status_code: int = 200,
        response_data: Optional[Dict[str, Any]] = None,
        headers: Optional[Dict[str, str]] = None
    ):
        """Mock a get summoner by name endpoint."""
        if response_data is None:
            response_data = RiotAPIMockData.get_summoner_response(summoner_name=summoner_name)
            
        url_pattern = f"https://{region}.api.riotgames.com/lol/summoner/v4/summoners/by-name/{summoner_name}"
        
        mock_headers = {"Content-Type": "application/json"}
        if headers:
            mock_headers.update(headers)
            
        self.router.get(url_pattern).mock(
            return_value=httpx.Response(
                status_code=status_code,
                json=response_data,
                headers=mock_headers
            )
        )
        
    def mock_summoner_not_found(self, summoner_name: str, region: str = "na1"):
        """Mock a summoner not found response (404)."""
        self.mock_get_summoner_by_name(
            summoner_name=summoner_name,
            region=region,
            status_code=404,
            response_data=RiotAPIMockData.get_error_response(404, "Summoner not found")
        )
        
    def mock_rate_limit_error(self, summoner_name: str, region: str = "na1", retry_after: int = 60):
        """Mock a rate limit error response (429)."""
        self.mock_get_summoner_by_name(
            summoner_name=summoner_name,
            region=region,
            status_code=429,
            response_data=RiotAPIMockData.get_error_response(429, "Rate limit exceeded"),
            headers={"Retry-After": str(retry_after)}
        )
        
    def mock_api_error(self, summoner_name: str, region: str = "na1", status_code: int = 500):
        """Mock a general API error response."""
        self.mock_get_summoner_by_name(
            summoner_name=summoner_name,
            region=region,
            status_code=status_code,
            response_data=RiotAPIMockData.get_error_response(status_code, "Internal server error")
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
    summoner_name: str = "TestSummoner",
    region: str = "na1",
    **summoner_data
):
    """Create a mock for successful summoner lookup."""
    router = RiotAPIMockRouter()
    
    response_data = RiotAPIMockData.get_summoner_response(
        summoner_name=summoner_name, **summoner_data
    )
    router.mock_get_summoner_by_name(
        summoner_name=summoner_name,
        region=region,
        response_data=response_data
    )
    
    return router


def mock_summoner_not_found(summoner_name: str = "NonExistentSummoner", region: str = "na1"):
    """Create a mock for summoner not found."""
    router = RiotAPIMockRouter()
    router.mock_summoner_not_found(summoner_name, region)
    return router


def mock_rate_limit_error(summoner_name: str = "TestSummoner", region: str = "na1", retry_after: int = 60):
    """Create a mock for rate limit error."""
    router = RiotAPIMockRouter()
    router.mock_rate_limit_error(summoner_name, region, retry_after)
    return router


def mock_api_error(summoner_name: str = "TestSummoner", region: str = "na1", status_code: int = 500):
    """Create a mock for API error."""
    router = RiotAPIMockRouter()
    router.mock_api_error(summoner_name, region, status_code)
    return router


# Legacy mock support using Mock objects (for compatibility with existing tests)
def create_mock_riot_api_client():
    """Create a Mock RiotAPIClient for unit testing."""
    mock_client = Mock()
    
    # Configure default successful response
    mock_client.get_summoner_by_name.return_value = RiotAPIMockData.get_summoner_response()
    
    return mock_client