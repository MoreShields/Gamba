"""Riot API client with rate limiting and error handling."""

import asyncio
import time
from typing import Optional, Dict, Any
from dataclasses import dataclass
from enum import Enum

import httpx
import structlog

logger = structlog.get_logger()


class RiotRegion:
    """Riot API region validation and mapping."""
    
    VALID_REGIONS = {
        "br1", "eun1", "euw1", "jp1", "kr", "la1", "la2",
        "na1", "oc1", "tr1", "ru", "ph2", "sg2", "th2", "tw2", "vn2"
    }
    
    @classmethod
    def is_valid(cls, region: str) -> bool:
        """Check if a region is valid."""
        return region.lower() in cls.VALID_REGIONS


@dataclass
class SummonerInfo:
    """Summoner information from Riot API."""

    puuid: str
    game_name: str  # Game name without tag
    tag_line: str  # Tag line is now required


@dataclass
class CurrentGameInfo:
    """Current game information from Spectator API."""

    game_id: str
    game_type: str
    game_start_time: int
    map_id: int
    game_length: int
    platform_id: str
    game_mode: str
    game_queue_config_id: int
    participants: list[Dict[str, Any]]

    @property
    def queue_type(self) -> str:
        """Get a readable queue type name."""
        queue_map = {
            420: "RANKED_SOLO_5x5",
            440: "RANKED_FLEX_SR",
            450: "ARAM",
            400: "NORMAL_DRAFT",
            430: "NORMAL_BLIND",
            700: "CLASH",
            1700: "ARENA",
        }
        return queue_map.get(
            self.game_queue_config_id, f"QUEUE_{self.game_queue_config_id}"
        )


@dataclass
class MatchInfo:
    """Match information from Match API."""

    match_id: str
    game_creation: int
    game_duration: int
    game_end_timestamp: int
    game_mode: str
    game_type: str
    map_id: int
    platform_id: str
    queue_id: int
    participants: list[Dict[str, Any]]

    @property
    def queue_type(self) -> str:
        """Get a readable queue type name."""
        queue_map = {
            420: "RANKED_SOLO_5x5",
            440: "RANKED_FLEX_SR",
            450: "ARAM",
            400: "NORMAL_DRAFT",
            430: "NORMAL_BLIND",
            700: "CLASH",
            1700: "ARENA",
        }
        return queue_map.get(self.queue_id, f"QUEUE_{self.queue_id}")

    def get_participant_result(self, puuid: str) -> Optional[Dict[str, Any]]:
        """Get the match result for a specific participant by PUUID."""
        for participant in self.participants:
            if participant.get("puuid") == puuid:
                return {
                    "won": participant.get("win", False),
                    "champion_name": participant.get("championName", "Unknown"),
                    "champion_id": participant.get("championId", 0),
                    "kills": participant.get("kills", 0),
                    "deaths": participant.get("deaths", 0),
                    "assists": participant.get("assists", 0),
                }
        return None


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


class PlayerNotInGameError(RiotAPIError):
    """Player is not currently in a game."""

    pass


class RiotAPIClient:
    """Riot API client with rate limiting and error handling."""

    def __init__(self, api_key: str, base_url: Optional[str] = None, request_timeout: float = 10.0):
        """Initialize the Riot API client.

        Args:
            api_key: Riot API key
            base_url: Base URL for the API (defaults to production Riot API)
            request_timeout: Request timeout in seconds
        """
        self.api_key = api_key
        self.base_url = base_url
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
        # Always use the provided base URL if available
        return self.base_url if self.base_url else f"https://{region}.api.riotgames.com"

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

    async def _make_request(
        self, url: str, handle_404_as: str = "summoner_not_found"
    ) -> Dict[str, Any]:
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

            # Handle not found - context dependent
            if response.status_code == 404:
                if handle_404_as == "summoner_not_found":
                    raise SummonerNotFoundError("Summoner not found")
                elif handle_404_as == "account_not_found":
                    raise SummonerNotFoundError("Account not found")
                elif handle_404_as == "not_in_game":
                    raise PlayerNotInGameError("Player is not currently in a game")
                else:
                    raise RiotAPIError("Resource not found")

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

    async def get_summoner_by_name(self, game_name: str, tag_line: str) -> SummonerInfo:
        """Get summoner information by summoner name.

        This uses only the account API to get basic summoner info with PUUID.

        Args:
            game_name: The game name part of the summoner name
            tag_line: The tag line part of the summoner name

        Returns:
            SummonerInfo object with basic summoner details (mainly PUUID)

        Raises:
            SummonerNotFoundError: If summoner is not found
            RiotAPIError: For other API errors
        """
        # Validate arguments
        if not game_name or not game_name.strip():
            raise SummonerNotFoundError("Game name cannot be empty")
        
        if not tag_line or not tag_line.strip():
            raise SummonerNotFoundError("Tag line cannot be empty")
        
        # Clean up the inputs
        game_name = game_name.strip()
        tag_line = tag_line.strip()

        logger.info(
            "Fetching summoner by name",
            game_name=game_name,
            tag_line=tag_line,
        )

        try:
            # Get account info by Riot ID
            summoner_info = await self.get_account_by_riot_id(game_name, tag_line)

            logger.info(
                "Successfully fetched summoner",
                game_name=summoner_info.game_name,
                puuid=summoner_info.puuid,
            )

            return summoner_info

        except SummonerNotFoundError:
            logger.info("Summoner not found", game_name=game_name, tag_line=tag_line)
            raise SummonerNotFoundError(f"Summoner not found: {game_name}#{tag_line}")
        except Exception as e:
            logger.error(
                "Error fetching summoner",
                game_name=game_name,
                tag_line=tag_line,
                error=str(e),
            )
            raise

    async def get_current_game_info(
        self, puuid: str, game_name: str, tag_line: str
    ) -> CurrentGameInfo:
        """Get current game information for a summoner by PUUID.

        Args:
            puuid: Player's PUUID
            region: Region to search in

        Returns:
            CurrentGameInfo object with current game details

        Raises:
            PlayerNotInGameError: If player is not currently in a game
            InvalidRegionError: If region is invalid
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """

        base_url = self._get_base_url(tag_line)
        # Try spectator v5 with PUUID first, fall back to v4 if needed
        url = f"{base_url}/lol/spectator/v5/active-games/by-summoner/{puuid}"

        try:
            data = await self._make_request(url, handle_404_as="not_in_game")

            current_game = CurrentGameInfo(
                game_id=str(data["gameId"]),
                game_type=data["gameType"],
                game_start_time=data["gameStartTime"],
                map_id=data["mapId"],
                game_length=data["gameLength"],
                platform_id=data["platformId"],
                game_mode=data["gameMode"],
                game_queue_config_id=data["gameQueueConfigId"],
                participants=data.get("participants", []),
            )

            return current_game

        except PlayerNotInGameError:
            raise
        except Exception as e:
            logger.error(
                "Error fetching current game info",
                game_name=game_name,
                puuid=puuid,
                tag_line=tag_line,
                error=str(e),
            )
            raise

    async def get_account_by_riot_id(
        self, game_name: str, tag_line: str
    ) -> SummonerInfo:
        """Get account information by Riot ID (game name and tag line).

        Args:
            game_name: Game name part of Riot ID
            tag_line: Tag line part of Riot ID (without #)

        Returns:
            SummonerInfo object with account details

        Raises:
            SummonerNotFoundError: If account is not found
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        # Account API uses americas endpoint, or custom base URL if provided
        base_url = self.base_url if self.base_url else "https://americas.api.riotgames.com"
        url = f"{base_url}/riot/account/v1/accounts/by-riot-id/{game_name}/{tag_line}"

        logger.info(
            "Fetching account by Riot ID",
            game_name=game_name,
            tag_line=tag_line,
        )

        try:
            data = await self._make_request(url, handle_404_as="account_not_found")

            summoner_info = SummonerInfo(
                puuid=data["puuid"],
                game_name=data["gameName"],
                tag_line=data["tagLine"],
            )

            logger.info(
                "Successfully fetched account",
                game_name=summoner_info.game_name,
                tag_line=summoner_info.tag_line,
                puuid=summoner_info.puuid,
            )

            return summoner_info

        except SummonerNotFoundError:
            logger.info(
                "Account not found",
                game_name=game_name,
                tag_line=tag_line,
            )
            raise
        except Exception as e:
            logger.error(
                "Error fetching account",
                game_name=game_name,
                tag_line=tag_line,
                error=str(e),
            )
            raise


    def _get_regional_url(self, region: str) -> str:
        """Get the regional URL for Match API calls."""
        # If custom base URL is provided, use it directly
        if self.base_url:
            return self.base_url
            
        # Otherwise use regional endpoints for production
        regional_map = {
            "na1": "americas",
            "br1": "americas",
            "la1": "americas",
            "la2": "americas",
            "euw1": "europe",
            "eun1": "europe",
            "tr1": "europe",
            "ru": "europe",
            "kr": "asia",
            "jp1": "asia",
            "oc1": "sea",
        }
        regional_endpoint = regional_map.get(region, "americas")
        return f"https://{regional_endpoint}.api.riotgames.com"

    async def get_match_info(self, match_id: str, region: str) -> MatchInfo:
        """Get detailed match information.

        Args:
            match_id: Match ID to fetch
            region: Region for routing (used to determine regional endpoint)

        Returns:
            MatchInfo object with match details

        Raises:
            InvalidRegionError: If region is invalid
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        # Skip region validation for now - regional mapping handles unknown regions

        regional_url = self._get_regional_url(region)
        url = f"{regional_url}/lol/match/v5/matches/{match_id}"

        logger.info("Fetching match info", match_id=match_id, region=region)

        try:
            data = await self._make_request(url)

            match_info = MatchInfo(
                match_id=data["metadata"]["matchId"],
                game_creation=data["info"]["gameCreation"],
                game_duration=data["info"]["gameDuration"],
                game_end_timestamp=data["info"]["gameEndTimestamp"],
                game_mode=data["info"]["gameMode"],
                game_type=data["info"]["gameType"],
                map_id=data["info"]["mapId"],
                platform_id=data["info"]["platformId"],
                queue_id=data["info"]["queueId"],
                participants=data["info"]["participants"],
            )

            logger.info(
                "Successfully fetched match info",
                match_id=match_id,
                region=region,
                queue_type=match_info.queue_type,
                duration=match_info.game_duration,
            )

            return match_info

        except Exception as e:
            logger.error(
                "Error fetching match info",
                match_id=match_id,
                region=region,
                error=str(e),
            )
            raise

    async def get_recent_matches(
        self, puuid: str, region: str, count: int = 5
    ) -> list[str]:
        """Get recent match IDs for a player.

        Args:
            puuid: Player PUUID
            region: Region for routing
            count: Number of matches to retrieve (max 100)

        Returns:
            List of match IDs

        Raises:
            InvalidRegionError: If region is invalid
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        # Skip region validation for now - regional mapping handles unknown regions

        regional_url = self._get_regional_url(region)
        url = f"{regional_url}/lol/match/v5/matches/by-puuid/{puuid}/ids"

        # Add query parameters
        params = {"count": min(count, 100)}

        logger.info("Fetching recent matches", puuid=puuid, region=region, count=count)

        try:
            response = await self.client.get(
                url,
                headers={"X-Riot-Token": self.api_key, "Accept": "application/json"},
                params=params,
            )

            if response.status_code == 404:
                logger.info("No matches found for player", puuid=puuid, region=region)
                return []

            if response.status_code >= 400:
                logger.error(
                    "Error fetching recent matches",
                    status_code=response.status_code,
                    response=response.text,
                )
                raise RiotAPIError(f"API error: {response.status_code}")

            match_ids = response.json()
            logger.info(
                "Successfully fetched recent matches",
                puuid=puuid,
                region=region,
                match_count=len(match_ids),
            )

            return match_ids

        except httpx.RequestError as e:
            logger.error("HTTP request failed", error=str(e))
            raise RiotAPIError(f"Request failed: {e}")
        except Exception as e:
            logger.error(
                "Error fetching recent matches",
                puuid=puuid,
                region=region,
                error=str(e),
            )
            raise
