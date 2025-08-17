"""Riot API client with rate limiting and error handling."""

import asyncio
import time
from typing import Optional, Dict, Any, Union, Tuple
from dataclasses import dataclass

from lol_tracker.core.enums import QueueType

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
        qt = QueueType.from_queue_id(self.game_queue_config_id)
        return qt.value if qt else f"QUEUE_{self.game_queue_config_id}"
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dict format expected by transition service."""
        return {
            "gameId": int(self.game_id),
            "gameQueueConfigId": self.game_queue_config_id,
            "gameStartTime": self.game_start_time,
            "gameLength": self.game_length,
            "gameType": self.game_type,
            "gameMode": self.game_mode,
            "mapId": self.map_id,
            "platformId": self.platform_id,
            "participants": self.participants,
            "game_type": "LOL"  # Marker for polymorphism
        }


@dataclass
class CurrentTFTGameInfo:
    """Current TFT game information from TFT Spectator API."""

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
        qt = QueueType.from_queue_id(self.game_queue_config_id)
        return qt.value if qt else f"QUEUE_{self.game_queue_config_id}"
    
    def to_dict(self) -> Dict[str, Any]:
        """Convert to dict format expected by transition service."""
        return {
            "gameId": int(self.game_id),
            "gameQueueConfigId": self.game_queue_config_id,
            "gameStartTime": self.game_start_time,
            "gameLength": self.game_length,
            "gameType": self.game_type,
            "gameMode": self.game_mode,
            "mapId": self.map_id,
            "platformId": self.platform_id,
            "participants": self.participants,
            "game_type": "TFT"  # Marker for polymorphism
        }


@dataclass
class TFTMatchInfo:
    """TFT match information from TFT Match API."""

    match_id: str
    game_creation: int
    game_datetime: int
    game_length: float
    game_variation: Optional[str]
    game_version: str
    participants: list[Dict[str, Any]]
    queue_id: int
    tft_game_type: Optional[str]
    tft_set_number: Optional[int]

    # Note: get_placement(puuid) removed - use get_placement_by_name instead
    
    def get_placement_by_name(self, game_name: str, tag_line: str) -> Optional[int]:
        """Get the player's placement (1-8) for the given Riot ID.
        
        Note: This requires participants to have riotIdGameName and riotIdTagline fields,
        which may not always be available. Falls back to None if not found.
        """
        for participant in self.participants:
            if (participant.get("riotIdGameName", "").lower() == game_name.lower() and 
                participant.get("riotIdTagline", "").lower() == tag_line.lower()):
                return participant.get("placement")
        return None


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

    # Note: get_participant_result(puuid) removed - use get_participant_result_by_name instead
    
    def get_participant_result_by_name(self, game_name: str, tag_line: str) -> Optional[Dict[str, Any]]:
        """Get the match result for a specific participant by Riot ID.
        
        Note: This requires participants to have riotIdGameName and riotIdTagline fields,
        which may not always be available. Falls back to None if not found.
        """
        for participant in self.participants:
            if (participant.get("riotIdGameName", "").lower() == game_name.lower() and 
                participant.get("riotIdTagline", "").lower() == tag_line.lower()):
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

    def __init__(self, api_key: str, tft_api_key: str, base_url: Optional[str] = None, request_timeout: float = 10.0):
        """Initialize the Riot API client.

        Args:
            api_key: Riot API key for LoL endpoints
            tft_api_key: Riot API key for TFT endpoints
            base_url: Base URL for the API (defaults to production Riot API)
            request_timeout: Request timeout in seconds
        """
        self.api_key = api_key
        self.tft_api_key = tft_api_key
        self.base_url = base_url
        self.request_timeout = request_timeout
        self.client = httpx.AsyncClient(timeout=request_timeout)

        # Simple rate limiting - track last request time
        self._last_request_time = 0.0
        self._min_request_interval = 1.2  # 1.2 seconds between requests to be safe

        # Rate limit tracking for 429 responses
        self._rate_limit_reset_time = 0.0
        
        # PUUID cache: {(game_name, tag_line, api_key_type): puuid}
        # api_key_type is 'lol' or 'tft' to differentiate between keys
        self._puuid_cache: Dict[Tuple[str, str, str], str] = {}

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
            # Rate limiting: wait before next request
            await asyncio.sleep(wait_time)

        self._last_request_time = time.time()

    async def _make_request(
        self, url: str, handle_404_as: str = "summoner_not_found", use_tft_key: bool = False
    ) -> Dict[str, Any]:
        """Make a request to the Riot API with rate limiting and error handling.
        
        Args:
            url: The URL to request
            handle_404_as: How to handle 404 responses
            use_tft_key: Whether to use the TFT API key instead of the main key
        """
        await self._rate_limit_delay()

        api_key = self.tft_api_key if use_tft_key else self.api_key
        headers = {"X-Riot-Token": api_key, "Accept": "application/json"}

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
                elif handle_404_as == "match_not_found":
                    raise RiotAPIError("Match not found")
                else:
                    raise RiotAPIError("Resource not found")

            # Handle other errors
            if response.status_code >= 400:
                logger.error(
                    "Riot API error",
                    url=url,
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
        except Exception:
            raise

    async def _get_puuid_for_key(self, game_name: str, tag_line: str, use_tft_key: bool) -> str:
        """Get PUUID for a player using the appropriate API key.
        
        Caches PUUIDs per API key to avoid repeated lookups.
        
        Args:
            game_name: Player's game name
            tag_line: Player's tag line
            use_tft_key: Whether to use TFT API key (True) or LoL API key (False)
            
        Returns:
            Player's PUUID for the specified API key
            
        Raises:
            SummonerNotFoundError: If player cannot be found
        """
        cache_key = (game_name, tag_line, 'tft' if use_tft_key else 'lol')
        
        # Check cache first
        if cache_key in self._puuid_cache:
            return self._puuid_cache[cache_key]
        
        # Fetch PUUID with appropriate key
        # We need to make the request with the correct API key
        summoner_info = await self._get_account_by_riot_id_with_key(game_name, tag_line, use_tft_key)
        puuid = summoner_info.puuid
        
        # Cache the result
        self._puuid_cache[cache_key] = puuid
        
        return puuid
    
    async def _get_account_by_riot_id_with_key(
        self, game_name: str, tag_line: str, use_tft_key: bool
    ) -> SummonerInfo:
        """Get account information by Riot ID using a specific API key.
        
        Internal method that allows specifying which API key to use.
        """
        # Account API ALWAYS uses americas endpoint for production
        # Only use base_url for test/mock environments
        if self.base_url and ("localhost" in self.base_url or "mock" in self.base_url):
            # Use provided base_url for test/mock environments
            base_url = self.base_url
        else:
            # Always use americas endpoint for production, regardless of configured base_url
            # Account API is global and doesn't use regional endpoints
            base_url = "https://americas.api.riotgames.com"
        url = f"{base_url}/riot/account/v1/accounts/by-riot-id/{game_name}/{tag_line}"

        logger.info(
            "Fetching account by Riot ID",
            url=url,
            use_tft_key=use_tft_key,
            game_name=game_name,
            tag_line=tag_line,
        )
        
        try:
            data = await self._make_request(url, handle_404_as="account_not_found", use_tft_key=use_tft_key)
            
            summoner_info = SummonerInfo(
                puuid=data["puuid"],
                game_name=data["gameName"],
                tag_line=data["tagLine"],
            )

            logger.info(
                "Successfully fetched account",
                use_tft_key=use_tft_key,
                game_name=summoner_info.game_name,
                tag_line=summoner_info.tag_line,
                puuid=summoner_info.puuid,
            )

            return summoner_info
            
        except Exception:
            raise
    
    async def get_current_lol_game_info(
        self, game_name: str, tag_line: str
    ) -> CurrentGameInfo:
        """Get current LoL game information for a summoner.

        Args:
            game_name: Player's game name
            tag_line: Player's tag line

        Returns:
            CurrentGameInfo object with current game details

        Raises:
            PlayerNotInGameError: If player is not currently in a game
            SummonerNotFoundError: If player cannot be found
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        # Get PUUID with LoL API key
        puuid = await self._get_puuid_for_key(game_name, tag_line, use_tft_key=False)

        # Use na1 region for spectator API (hardcoded for now, could be made configurable)
        base_url = self._get_base_url("na1")
        # Try spectator v5 with PUUID first, fall back to v4 if needed
        url = f"{base_url}/lol/spectator/v5/active-games/by-summoner/{puuid}"

        try:
            data = await self._make_request(url, handle_404_as="not_in_game", use_tft_key=False)

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
            logger.warning(f"Failed to get LoL game info for {game_name}#{tag_line}: {e}")
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
        try:
            # Delegate to the internal method with the default LoL API key
            return await self._get_account_by_riot_id_with_key(game_name, tag_line, use_tft_key=False)
        except SummonerNotFoundError:
            logger.info(
                "Account not found",
                game_name=game_name,
                tag_line=tag_line,
            )
            raise
        except Exception:
            raise


    def _get_regional_url(self, region: str) -> str:
        """Get the regional URL for Match API calls."""
        # For test/mock environments, use the base_url
        if self.base_url and ("localhost" in self.base_url or "mock" in self.base_url):
            return self.base_url
        
        # For production, match endpoints ALWAYS use regional routing
        # regardless of any configured base_url
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
            data = await self._make_request(url, handle_404_as="match_not_found")

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

        except Exception:
            raise


    async def get_current_tft_game_info(
        self, game_name: str, tag_line: str
    ) -> CurrentTFTGameInfo:
        """Get current TFT game information for a summoner.

        Args:
            game_name: Player's game name
            tag_line: Player's tag line

        Returns:
            CurrentTFTGameInfo object with current TFT game details

        Raises:
            PlayerNotInGameError: If player is not currently in a TFT game
            SummonerNotFoundError: If player cannot be found
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        # Get PUUID with TFT API key
        puuid = await self._get_puuid_for_key(game_name, tag_line, use_tft_key=True)
        # Use na1 region for TFT spectator API (hardcoded for now, could be made configurable)
        base_url = self._get_base_url("na1")
        url = f"{base_url}/lol/spectator/tft/v5/active-games/by-puuid/{puuid}"

        try:
            data = await self._make_request(url, handle_404_as="not_in_game", use_tft_key=True)

            current_game = CurrentTFTGameInfo(
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
            logger.warning(f"Failed to get TFT game info for {game_name}#{tag_line}: {e}")
            raise

    async def get_tft_match_info(self, match_id: str) -> TFTMatchInfo:
        """Get detailed TFT match information.

        Args:
            match_id: TFT Match ID to fetch

        Returns:
            TFTMatchInfo object with match details

        Raises:
            RateLimitError: If rate limited
            RiotAPIError: For other API errors
        """
        # TFT Match API uses regional routing like LoL
        # For now, hardcode to na1 region (which maps to americas)
        regional_url = self._get_regional_url("na1")
        url = f"{regional_url}/tft/match/v1/matches/{match_id}"

        logger.info("Fetching TFT match info", match_id=match_id)

        try:
            data = await self._make_request(url, use_tft_key=True)

            match_info = TFTMatchInfo(
                match_id=data["metadata"]["match_id"],
                game_creation=data["info"]["game_datetime"],
                game_datetime=data["info"]["game_datetime"],
                game_length=data["info"]["game_length"],
                game_variation=data["info"].get("game_variation"),
                game_version=data["info"]["game_version"],
                participants=data["info"]["participants"],
                queue_id=data["info"]["queue_id"],
                tft_game_type=data["info"].get("tft_game_type"),
                tft_set_number=data["info"].get("tft_set_number"),
            )

            logger.info(
                "Successfully fetched TFT match info",
                match_id=match_id,
                queue_id=match_info.queue_id,
                duration=match_info.game_length,
            )

            return match_info

        except Exception:
            raise

    async def get_match_for_game(
        self, game_id: str, game_type: str, region: str = "na1"
    ) -> Optional[Union["MatchInfo", "TFTMatchInfo"]]:
        """Get match information for a completed game.
        
        Args:
            game_id: The game ID to fetch
            game_type: 'LOL' or 'TFT'
            region: Region for API calls (default: "na1")
            
        Returns:
            MatchInfo for LoL, TFTMatchInfo for TFT, or None if not found
            
        Raises:
            RiotAPIError: For API errors
        """
        # Convert game_id to match_id format (both LoL and TFT use same format)
        match_id = f"{region.upper()}_{game_id}"
        
        try:
            # Call appropriate API based on game type
            if game_type == 'LOL':
                return await self.get_match_info(match_id, region)
            elif game_type == 'TFT':
                return await self.get_tft_match_info(match_id)
            else:
                logger.warning(f"Unknown game type: {game_type}")
                return None
        except RiotAPIError as e:
            # If it's a 404, the match might not be available yet
            if "404" in str(e) or "not found" in str(e).lower():
                return None
            raise
    
    async def get_active_game_info(
        self, game_name: str, tag_line: str
    ) -> Optional[Union[CurrentGameInfo, CurrentTFTGameInfo]]:
        """Get active game info by checking both LoL and TFT endpoints in parallel.
        
        Checks both LoL and TFT endpoints concurrently and returns the first
        successful result (or None if both indicate player is not in game).
        
        Args:
            game_name: Player's game name  
            tag_line: Player's tag line
            
        Returns:
            CurrentGameInfo or CurrentTFTGameInfo if player is in a game, None otherwise
            
        Raises:
            SummonerNotFoundError: If player cannot be found
            RateLimitError: If rate limited
            RiotAPIError: For other API errors (but not PlayerNotInGameError)
        """
        # Check both LoL and TFT endpoints in parallel
        results = await asyncio.gather(
            self.get_current_lol_game_info(game_name, tag_line),
            self.get_current_tft_game_info(game_name, tag_line),
            return_exceptions=True
        )
        
        # Return first successful result
        for result in results:
            if not isinstance(result, Exception):
                return result
        
        # Re-raise any non-PlayerNotInGameError exceptions
        for result in results:
            if isinstance(result, Exception) and not isinstance(result, PlayerNotInGameError):
                raise result
        
        # Both endpoints returned PlayerNotInGameError - player is not in any game
        return None
