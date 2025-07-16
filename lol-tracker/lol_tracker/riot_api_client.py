"""Riot API client with rate limiting and error handling."""

import asyncio
import time
from typing import Optional, Dict, Any
from dataclasses import dataclass
from enum import Enum

import httpx
import structlog

logger = structlog.get_logger()


class RiotRegion(Enum):
    """Supported Riot API regions."""

    NA1 = "na1"
    EUW1 = "euw1"
    EUN1 = "eun1"
    KR = "kr"
    BR1 = "br1"
    LA1 = "la1"
    LA2 = "la2"
    OC1 = "oc1"
    RU = "ru"
    TR1 = "tr1"
    JP1 = "jp1"

    @classmethod
    def is_valid(cls, region: str) -> bool:
        """Check if a region string is valid."""
        return region in {r.value for r in cls}


@dataclass
class SummonerInfo:
    """Summoner information from Riot API."""

    puuid: str
    account_id: str
    summoner_id: str
    summoner_name: str
    summoner_level: int
    region: str
    last_updated: int


class RiotAPIError(Exception):
    """Base exception for Riot API errors."""

    pass


class SummonerNotFoundError(RiotAPIError):
    """Summoner not found error."""

    pass


class RateLimitError(RiotAPIError):
    """Rate limit exceeded error."""

    pass


class InvalidRegionError(RiotAPIError):
    """Invalid region error."""

    pass


class RiotAPIClient:
    """Riot API client with rate limiting and error handling."""

    def __init__(self, api_key: str, request_timeout: float = 10.0):
        """Initialize the Riot API client.

        Args:
            api_key: Riot API key
            request_timeout: Request timeout in seconds
        """
        self.api_key = api_key
        self.request_timeout = request_timeout
        self.client = httpx.AsyncClient(timeout=request_timeout)

        # Simple rate limiting - track last request time
        self._last_request_time = 0.0
        self._min_request_interval = 1.2  # 1.2 seconds between requests to be safe

        # Rate limit tracking for 429 responses
        self._rate_limit_reset_time = 0.0

    async def __aenter__(self):
        """Async context manager entry."""
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self.client.aclose()

    async def close(self):
        """Close the HTTP client."""
        await self.client.aclose()

    def _get_base_url(self, region: str) -> str:
        """Get the base URL for a region."""
        return f"https://{region}.api.riotgames.com"

    async def _rate_limit_delay(self):
        """Apply rate limiting delay."""
        current_time = time.time()

        # Check if we're in a rate limit cooldown
        if current_time < self._rate_limit_reset_time:
            wait_time = self._rate_limit_reset_time - current_time
            logger.info("Rate limit cooldown active", wait_time=wait_time)
            await asyncio.sleep(wait_time)

        # Apply minimum interval between requests
        time_since_last = current_time - self._last_request_time
        if time_since_last < self._min_request_interval:
            wait_time = self._min_request_interval - time_since_last
            await asyncio.sleep(wait_time)

        self._last_request_time = time.time()

    async def _make_request(self, url: str) -> Dict[str, Any]:
        """Make a request to the Riot API with rate limiting and error handling."""
        await self._rate_limit_delay()

        headers = {"X-Riot-Token": self.api_key, "Accept": "application/json"}

        try:
            response = await self.client.get(url, headers=headers)

            # Handle rate limiting
            if response.status_code == 429:
                retry_after = int(response.headers.get("Retry-After", 60))
                self._rate_limit_reset_time = time.time() + retry_after
                logger.warning("Rate limited by Riot API", retry_after=retry_after)
                raise RateLimitError(f"Rate limited. Retry after {retry_after} seconds")

            # Handle not found
            if response.status_code == 404:
                raise SummonerNotFoundError("Summoner not found")

            # Handle other errors
            if response.status_code >= 400:
                logger.error(
                    "Riot API error",
                    status_code=response.status_code,
                    response=response.text,
                )
                raise RiotAPIError(f"API error: {response.status_code}")

            return response.json()

        except httpx.RequestError as e:
            logger.error("HTTP request failed", error=str(e))
            raise RiotAPIError(f"Request failed: {e}")

    async def get_summoner_by_name(
        self, summoner_name: str, region: str
    ) -> SummonerInfo:
        """Get summoner information by summoner name.

        Args:
            summoner_name: Summoner name to look up
            region: Region to search in

        Returns:
            SummonerInfo object with summoner details

        Raises:
            SummonerNotFoundError: If summoner is not found
            InvalidRegionError: If region is invalid
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        if not RiotRegion.is_valid(region):
            raise InvalidRegionError(f"Invalid region: {region}")

        base_url = self._get_base_url(region)
        url = f"{base_url}/lol/summoner/v4/summoners/by-name/{summoner_name}"

        logger.info(
            "Fetching summoner by name", summoner_name=summoner_name, region=region
        )

        try:
            data = await self._make_request(url)

            # Extract summoner information
            summoner_info = SummonerInfo(
                puuid=data["puuid"],
                account_id=data["accountId"],
                summoner_id=data["id"],
                summoner_name=data["name"],
                summoner_level=data["summonerLevel"],
                region=region,
                last_updated=data["revisionDate"],
            )

            logger.info(
                "Successfully fetched summoner",
                summoner_name=summoner_info.summoner_name,
                region=region,
                summoner_level=summoner_info.summoner_level,
            )

            return summoner_info

        except SummonerNotFoundError:
            logger.info(
                "Summoner not found", summoner_name=summoner_name, region=region
            )
            raise
        except Exception as e:
            logger.error(
                "Error fetching summoner",
                summoner_name=summoner_name,
                region=region,
                error=str(e),
            )
            raise

    async def validate_summoner_exists(self, summoner_name: str, region: str) -> bool:
        """Validate if a summoner exists without returning full details.

        Args:
            summoner_name: Summoner name to validate
            region: Region to search in

        Returns:
            True if summoner exists, False otherwise
        """
        try:
            await self.get_summoner_by_name(summoner_name, region)
            return True
        except SummonerNotFoundError:
            return False
        except Exception:
            # For other errors (rate limits, API errors), we should raise them
            raise
