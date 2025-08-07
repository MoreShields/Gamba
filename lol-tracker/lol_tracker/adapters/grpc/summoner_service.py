"""Simplified gRPC service implementation for summoner tracking."""

from datetime import datetime
from typing import Optional

import grpc
import structlog
from google.protobuf.timestamp_pb2 import Timestamp

from lol_tracker.proto.services import summoner_service_pb2
from lol_tracker.proto.services import summoner_service_pb2_grpc

from lol_tracker.adapters.riot_api.client import RiotAPIClient
from lol_tracker.adapters.riot_api import (
    SummonerNotFoundError,
    InvalidRegionError,
    RateLimitError,
    RiotAPIError,
)
from lol_tracker.adapters.database.manager import DatabaseManager

logger = structlog.get_logger()


class SummonerTrackingService(
    summoner_service_pb2_grpc.SummonerTrackingServiceServicer
):
    """Simplified gRPC service for summoner tracking with direct database access."""

    def __init__(self, db_manager: DatabaseManager, riot_api_service: RiotAPIClient):
        """Initialize the summoner tracking service.

        Args:
            db_manager: Database manager for direct repository access
            riot_api_service: Riot API client for external API calls
        """
        self.db_manager = db_manager
        self.riot_api_service = riot_api_service

    async def close(self):
        """Close the service and clean up resources."""
        # Note: We don't close riot_api_client here as it's managed externally
        pass

    async def StartTrackingSummoner(self, request, context):
        """Start tracking a summoner with immediate validation.

        Args:
            request: StartTrackingSummonerRequest
            context: gRPC context

        Returns:
            StartTrackingSummonerResponse
        """
        logger.info(
            "Starting summoner tracking",
            game_name=request.game_name,
        )

        try:
            # Validate required fields
            if not request.game_name or not request.tag_line:
                logger.error(
                    "Missing game name or tag line",
                    game_name=request.game_name,
                    tag_line=request.tag_line,
                )
                return summoner_service_pb2.StartTrackingSummonerResponse(
                    success=False,
                    error_message="Both game name and tag line are required",
                    error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_SUMMONER_NOT_FOUND,
                )
            
            # Validate summoner exists via Riot API
            summoner_info = await self.riot_api_service.get_summoner_by_name(
                request.game_name, request.tag_line
            )
            
            if summoner_info is None:
                raise SummonerNotFoundError(f"Summoner '{request.game_name}#{request.tag_line}' not found")

            # Check if summoner is already being tracked by Riot ID
            existing_player = await self.db_manager.get_tracked_player_by_riot_id(
                summoner_info.game_name,
                summoner_info.tag_line
            )

            if existing_player:
                logger.info(
                    "Summoner already being tracked",
                    game_name=request.game_name,
                    tag_line=request.tag_line
                )
                return summoner_service_pb2.StartTrackingSummonerResponse(
                    success=True,
                    summoner_details = summoner_service_pb2.SummonerDetails(
                        game_name=summoner_info.game_name,
                        tag_line=summoner_info.tag_line,
                        summoner_level=0,
                        last_updated=0,
                    )
                )

            # Create new tracked player using DatabaseManager
            # Note: We no longer store PUUID
            await self.db_manager.create_tracked_player(
                game_name=summoner_info.game_name,
                tag_line=summoner_info.tag_line
            )
            logger.info(
                "Created new tracked summoner",
                game_name=request.game_name,
                tag_line=request.tag_line
            )

            # Create summoner details response
            summoner_details = summoner_service_pb2.SummonerDetails(
                game_name=summoner_info.game_name,
                tag_line=summoner_info.tag_line,
                summoner_level=0,
                last_updated=0,
            )

            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=True, summoner_details=summoner_details
            )

        except SummonerNotFoundError:
            logger.info(
                "Summoner not found",
                game_name=request.game_name,
                tag_line=request.tag_line,
            )
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message=f"Summoner '{request.game_name}#{request.tag_line}' not found",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_SUMMONER_NOT_FOUND,
            )

        except InvalidRegionError as e:
            logger.warning("Invalid region error", error=str(e))
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message=str(e),
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_INVALID_REGION,
            )

        except RateLimitError as e:
            logger.warning("Rate limited by Riot API", error=str(e))
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message="Rate limited by Riot API. Please try again later.",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_RATE_LIMITED,
            )

        except RiotAPIError as e:
            logger.error("Riot API error", error=str(e))
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message=f"Riot API error: {str(e)}",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_API_ERROR,
            )

        except Exception as e:
            logger.error("Internal error during summoner tracking", error=str(e))
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message="Internal service error. Please try again later.",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_INTERNAL_ERROR,
            )

    async def StopTrackingSummoner(self, request, context):
        """Stop tracking a summoner.

        Args:
            request: StopTrackingSummonerRequest
            context: gRPC context

        Returns:
            StopTrackingSummonerResponse
        """
        logger.info(
            "Stopping summoner tracking",
            game_name=request.game_name,
            tag_line=request.tag_line,
        )

        try:
            # Validate required fields
            if not request.game_name or not request.tag_line:
                logger.error(
                    "Missing game name or tag line",
                    game_name=request.game_name,
                    tag_line=request.tag_line,
                )
                return summoner_service_pb2.StopTrackingSummonerResponse(
                    success=False,
                    error_message="Both game name and tag line are required",
                    error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_SUMMONER_NOT_FOUND,
                )
            
            # First, get the summoner info to get the PUUID
            summoner_info = await self.riot_api_service.get_summoner_by_name(
                request.game_name, request.tag_line
            )
            
            if summoner_info is None:
                raise SummonerNotFoundError(f"Summoner '{request.game_name}#{request.tag_line}' not found")
            
            # Find the tracked player by Riot ID
            tracked_player = await self.db_manager.get_tracked_player_by_riot_id(
                summoner_info.game_name,
                summoner_info.tag_line
            )

            if not tracked_player:
                logger.info(
                    "Summoner not currently being tracked",
                    game_name=request.game_name,
                    tag_line=request.tag_line,
                )
                return summoner_service_pb2.StopTrackingSummonerResponse(
                    success=False,
                    error_message=f"Summoner {request.game_name}#{request.tag_line} is not currently being tracked",
                    error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_NOT_TRACKED,
                )

            # Delete the tracked player entirely using DatabaseManager
            await self.db_manager.delete_tracked_player(tracked_player.id)

            logger.info(
                "Successfully stopped tracking summoner",
                game_name=request.game_name,
                tag_line=request.tag_line,
            )

            return summoner_service_pb2.StopTrackingSummonerResponse(success=True)

        except Exception as e:
            logger.error("Internal error during summoner untracking", error=str(e))
            return summoner_service_pb2.StopTrackingSummonerResponse(
                success=False,
                error_message="Internal service error. Please try again later.",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_INTERNAL_ERROR,
            )