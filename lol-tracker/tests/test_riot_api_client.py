"""Tests for Riot API client."""

import asyncio
import pytest
from unittest.mock import Mock, patch
import httpx

from lol_tracker.riot_api import (
    RiotAPIClient,
    SummonerInfo,
    SummonerNotFoundError,
    InvalidRegionError,
    RateLimitError,
    RiotAPIError,
)

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
        assert client._get_base_url("NA1") == "https://NA1.api.riotgames.com"
        assert client._get_base_url("EUW1") == "https://EUW1.api.riotgames.com"

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_invalid_format(self):
        """Test that empty game name or tag line raises SummonerNotFoundError."""
        client = RiotAPIClient("test_api_key")

        with pytest.raises(SummonerNotFoundError):
            await client.get_summoner_by_name("", "tag")
        
        with pytest.raises(SummonerNotFoundError):
            await client.get_summoner_by_name("name", "")
        
        with pytest.raises(SummonerNotFoundError):
            await client.get_summoner_by_name("  ", "tag")

        await client.close()

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_success(self):
        """Test successful summoner lookup using account API only."""
        mock_summoner_info = SummonerInfo(
            puuid="puuid_789",
            game_name="TestPlayer",
            tag_line="gamba"
        )

        client = RiotAPIClient("test_api_key")

        with patch.object(client, "get_account_by_riot_id") as mock_get_account:
            mock_get_account.return_value = mock_summoner_info

            result = await client.get_summoner_by_name("TestPlayer", "gamba")

            assert isinstance(result, SummonerInfo)
            assert result.game_name == "TestPlayer"
            assert result.puuid == "puuid_789"
            assert result.tag_line == "gamba"

            # Verify the correct method was called
            mock_get_account.assert_called_once_with("TestPlayer", "gamba")

        await client.close()

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_not_found(self):
        """Test summoner not found scenario - account not found."""
        client = RiotAPIClient("test_api_key")

        with patch.object(client, "get_account_by_riot_id") as mock_get_account:
            mock_get_account.side_effect = SummonerNotFoundError("Account not found")

            with pytest.raises(SummonerNotFoundError):
                await client.get_summoner_by_name("NonExistentPlayer", "NA1")

        await client.close()

    @pytest.mark.asyncio
    async def test_get_summoner_by_name_with_tagline(self):
        """Test summoner lookup with explicit tagline."""
        mock_summoner_info = SummonerInfo(
            puuid="puuid_789",
            game_name="TestPlayer",
            tag_line="EUW"
        )

        client = RiotAPIClient("test_api_key")

        with patch.object(client, "get_account_by_riot_id") as mock_get_account:
            mock_get_account.return_value = mock_summoner_info

            result = await client.get_summoner_by_name("TestPlayer", "EUW")

            assert result.game_name == "TestPlayer"
            assert result.puuid == "puuid_789"

            # Verify the tagline was parsed correctly
            mock_get_account.assert_called_once_with("TestPlayer", "EUW")

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
        with patch("lol_tracker.riot_api.riot_api_client.time.time") as mock_time:
            with patch("lol_tracker.riot_api.riot_api_client.asyncio.sleep") as mock_sleep:
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
