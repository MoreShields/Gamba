"""Tests for Riot API client."""

import asyncio
import pytest
from unittest.mock import Mock, patch
import httpx

from lol_tracker.riot_api_client import (
    RiotAPIClient,
    RiotRegion,
    SummonerInfo,
    SummonerNotFoundError,
    InvalidRegionError,
    RateLimitError,
    RiotAPIError,
)


class TestRiotRegion:
    """Test cases for RiotRegion enum."""

    def test_valid_regions(self):
        """Test that valid regions are recognized."""
        assert RiotRegion.is_valid("na1")
        assert RiotRegion.is_valid("euw1")
        assert RiotRegion.is_valid("kr")

    def test_invalid_regions(self):
        """Test that invalid regions are rejected."""
        assert not RiotRegion.is_valid("invalid")
        assert not RiotRegion.is_valid("na2")
        assert not RiotRegion.is_valid("europe")


class TestRiotAPIClient:
    """Test cases for RiotAPIClient."""

    def test_initialization(self):
        """Test client initialization."""
        client = RiotAPIClient("test_api_key")
        assert client.api_key == "test_api_key"
        assert client.request_timeout == 10.0
        assert client._min_request_interval == 1.2

    def test_initialization_with_custom_timeout(self):
        """Test client initialization with custom timeout."""
        client = RiotAPIClient("test_api_key", request_timeout=30.0)
        assert client.request_timeout == 30.0

    def test_get_base_url(self):
        """Test base URL generation."""
        client = RiotAPIClient("test_api_key")
        assert client._get_base_url("na1") == "https://na1.api.riotgames.com"
        assert client._get_base_url("euw1") == "https://euw1.api.riotgames.com"

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_invalid_region(self):
        """Test that invalid region raises InvalidRegionError."""
        client = RiotAPIClient("test_api_key")

        with pytest.raises(InvalidRegionError):
            await client.get_summoner_by_name("TestPlayer", "invalid_region")

        await client.close()

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_success(self):
        """Test successful summoner lookup."""
        mock_response_data = {
            "id": "summoner_id_123",
            "accountId": "account_id_456",
            "puuid": "puuid_789",
            "name": "TestPlayer",
            "summonerLevel": 50,
            "revisionDate": 1640995200000,
        }

        client = RiotAPIClient("test_api_key")

        with patch.object(client, "_make_request") as mock_request:
            mock_request.return_value = mock_response_data

            result = await client.get_summoner_by_name("TestPlayer", "na1")

            assert isinstance(result, SummonerInfo)
            assert result.summoner_name == "TestPlayer"
            assert result.summoner_id == "summoner_id_123"
            assert result.account_id == "account_id_456"
            assert result.puuid == "puuid_789"
            assert result.summoner_level == 50
            assert result.region == "na1"
            assert result.last_updated == 1640995200000

            # Verify the correct URL was called
            mock_request.assert_called_once_with(
                "https://na1.api.riotgames.com/lol/summoner/v4/summoners/by-name/TestPlayer"
            )

        await client.close()

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_not_found(self):
        """Test summoner not found scenario."""
        client = RiotAPIClient("test_api_key")

        with patch.object(client, "_make_request") as mock_request:
            mock_request.side_effect = SummonerNotFoundError("Summoner not found")

            with pytest.raises(SummonerNotFoundError):
                await client.get_summoner_by_name("NonExistentPlayer", "na1")

        await client.close()

    @pytest.mark.asyncio
    async def test_validate_summoner_exists_true(self):
        """Test validate_summoner_exists returns True for existing summoner."""
        client = RiotAPIClient("test_api_key")

        with patch.object(client, "get_summoner_by_name") as mock_get:
            mock_get.return_value = SummonerInfo(
                puuid="test_puuid",
                account_id="test_account",
                summoner_id="test_summoner",
                summoner_name="TestPlayer",
                summoner_level=50,
                region="na1",
                last_updated=1640995200000,
            )

            result = await client.validate_summoner_exists("TestPlayer", "na1")
            assert result is True

        await client.close()

    @pytest.mark.asyncio
    async def test_validate_summoner_exists_false(self):
        """Test validate_summoner_exists returns False for non-existing summoner."""
        client = RiotAPIClient("test_api_key")

        with patch.object(client, "get_summoner_by_name") as mock_get:
            mock_get.side_effect = SummonerNotFoundError("Summoner not found")

            result = await client.validate_summoner_exists("NonExistentPlayer", "na1")
            assert result is False

        await client.close()

    @pytest.mark.asyncio
    async def test_validate_summoner_exists_propagates_other_errors(self):
        """Test that validate_summoner_exists propagates non-NotFound errors."""
        client = RiotAPIClient("test_api_key")

        with patch.object(client, "get_summoner_by_name") as mock_get:
            mock_get.side_effect = RateLimitError("Rate limited")

            with pytest.raises(RateLimitError):
                await client.validate_summoner_exists("TestPlayer", "na1")

        await client.close()

    @pytest.mark.asyncio
    async def test_make_request_success(self):
        """Test successful HTTP request."""
        client = RiotAPIClient("test_api_key")

        mock_response = Mock()
        mock_response.status_code = 200
        mock_response.json.return_value = {"test": "data"}

        with patch.object(client.client, "get") as mock_get:
            mock_get.return_value = mock_response

            result = await client._make_request("https://test.com/api")

            assert result == {"test": "data"}
            mock_get.assert_called_once_with(
                "https://test.com/api",
                headers={"X-Riot-Token": "test_api_key", "Accept": "application/json"},
            )

        await client.close()

    @pytest.mark.asyncio
    async def test_make_request_rate_limited(self):
        """Test rate limit handling."""
        client = RiotAPIClient("test_api_key")

        mock_response = Mock()
        mock_response.status_code = 429
        mock_response.headers = {"Retry-After": "60"}

        with patch.object(client.client, "get") as mock_get:
            mock_get.return_value = mock_response

            with pytest.raises(RateLimitError) as exc_info:
                await client._make_request("https://test.com/api")

            assert "Rate limited" in str(exc_info.value)
            assert "60 seconds" in str(exc_info.value)

        await client.close()

    @pytest.mark.asyncio
    async def test_make_request_not_found(self):
        """Test 404 response handling."""
        client = RiotAPIClient("test_api_key")

        mock_response = Mock()
        mock_response.status_code = 404

        with patch.object(client.client, "get") as mock_get:
            mock_get.return_value = mock_response

            with pytest.raises(SummonerNotFoundError):
                await client._make_request("https://test.com/api")

        await client.close()

    @pytest.mark.asyncio
    async def test_make_request_server_error(self):
        """Test server error response handling."""
        client = RiotAPIClient("test_api_key")

        mock_response = Mock()
        mock_response.status_code = 500
        mock_response.text = "Internal Server Error"

        with patch.object(client.client, "get") as mock_get:
            mock_get.return_value = mock_response

            with pytest.raises(RiotAPIError) as exc_info:
                await client._make_request("https://test.com/api")

            assert "API error: 500" in str(exc_info.value)

        await client.close()

    @pytest.mark.asyncio
    async def test_make_request_http_error(self):
        """Test HTTP request exception handling."""
        client = RiotAPIClient("test_api_key")

        with patch.object(client.client, "get") as mock_get:
            mock_get.side_effect = httpx.RequestError("Connection failed")

            with pytest.raises(RiotAPIError) as exc_info:
                await client._make_request("https://test.com/api")

            assert "Request failed" in str(exc_info.value)

        await client.close()

    @pytest.mark.asyncio
    async def test_rate_limit_delay(self):
        """Test rate limiting delay functionality."""
        client = RiotAPIClient("test_api_key")

        # Mock time to control timing
        with patch("lol_tracker.riot_api_client.time.time") as mock_time:
            with patch("lol_tracker.riot_api_client.asyncio.sleep") as mock_sleep:
                # First call - should delay because _last_request_time is 0
                mock_time.return_value = 0.0
                await client._rate_limit_delay()
                mock_sleep.assert_called_once_with(
                    1.2
                )  # Full interval since _last_request_time was 0

                # Reset mock
                mock_sleep.reset_mock()

                # Second call - simulate time passed since last request
                client._last_request_time = 1.0  # Set manually
                mock_time.side_effect = [
                    2.0,
                    2.0,
                ]  # Current time, then time for interval calc
                await client._rate_limit_delay()
                # Check that sleep was called with approximately 0.2 seconds
                mock_sleep.assert_called_once()
                call_args = mock_sleep.call_args[0]
                assert abs(call_args[0] - 0.2) < 0.001  # Within 1ms tolerance

        await client.close()

    @pytest.mark.asyncio
    async def test_context_manager(self):
        """Test async context manager functionality."""
        async with RiotAPIClient("test_api_key") as client:
            assert client.api_key == "test_api_key"
            # Client should be usable within context
            assert client.client is not None
