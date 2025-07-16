"""gRPC service implementation for summoner tracking."""

from datetime import datetime
from typing import Optional

import grpc
import structlog
from google.protobuf.timestamp_pb2 import Timestamp

from lol_tracker.proto.services import summoner_service_pb2
from lol_tracker.proto.services import summoner_service_pb2_grpc

from .riot_api_client import (
    RiotAPIClient,
    SummonerNotFoundError,
    InvalidRegionError,
    RateLimitError,
    RiotAPIError,
)
from .database.repository import TrackedPlayerRepository
from .database.connection import DatabaseManager

logger = structlog.get_logger()


class SummonerTrackingService(
    summoner_service_pb2_grpc.SummonerTrackingServiceServicer
):
    """gRPC service for summoner tracking with immediate validation."""

    def __init__(self, db_manager: DatabaseManager, riot_api_key: str):
        """Initialize the summoner tracking service.

        Args:
            db_manager: Database manager for repository access
            riot_api_key: Riot API key for summoner validation
        """
        self.db_manager = db_manager
        self.riot_api_client = RiotAPIClient(riot_api_key)

    async def close(self):
        """Close the service and clean up resources."""
        await self.riot_api_client.close()

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
            summoner_name=request.summoner_name,
            region=request.region,
        )

        try:
            # Validate summoner exists via Riot API
            summoner_info = await self.riot_api_client.get_summoner_by_name(
                request.summoner_name, request.region
            )

            # Use database session for repository operations
            async with self.db_manager.get_session() as session:
                tracked_player_repo = TrackedPlayerRepository(session)
                
                # Check if summoner is already being tracked
                existing_player = await tracked_player_repo.get_by_summoner_and_region(
                    summoner_info.summoner_name, request.region
                )

                if existing_player and existing_player.is_active:
                    logger.info(
                        "Summoner already being tracked",
                        summoner_name=request.summoner_name,
                        region=request.region,
                    )
                    return summoner_service_pb2.StartTrackingSummonerResponse(
                        success=False,
                        error_message=f"Summoner {request.summoner_name} is already being tracked in {request.region}",
                        error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_ALREADY_TRACKED,
                    )

                # Create or update tracked player record
                if existing_player:
                    # Reactivate existing player and update Riot IDs
                    await tracked_player_repo.update_riot_ids(
                        existing_player.id,
                        puuid=summoner_info.puuid,
                        account_id=summoner_info.account_id,
                        summoner_id=summoner_info.summoner_id,
                    )
                    await tracked_player_repo.set_active_status(existing_player.id, True)
                    logger.info(
                        "Reactivated existing summoner",
                        summoner_name=request.summoner_name,
                        region=request.region,
                    )
                else:
                    # Create new tracked player
                    await tracked_player_repo.create(
                        summoner_name=summoner_info.summoner_name,
                        region=request.region,
                        puuid=summoner_info.puuid,
                        account_id=summoner_info.account_id,
                        summoner_id=summoner_info.summoner_id,
                    )
                    logger.info(
                        "Created new tracked summoner",
                        summoner_name=request.summoner_name,
                        region=request.region,
                    )

                # Commit the transaction
                await session.commit()

            # Create summoner details response
            summoner_details = summoner_service_pb2.SummonerDetails(
                puuid=summoner_info.puuid,
                account_id=summoner_info.account_id,
                summoner_id=summoner_info.summoner_id,
                summoner_name=summoner_info.summoner_name,
                summoner_level=summoner_info.summoner_level,
                region=summoner_info.region,
                last_updated=summoner_info.last_updated,
            )

            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=True, summoner_details=summoner_details
            )

        except SummonerNotFoundError:
            logger.info(
                "Summoner not found",
                summoner_name=request.summoner_name,
                region=request.region,
            )
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message=f"Summoner '{request.summoner_name}' not found in region {request.region}",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_SUMMONER_NOT_FOUND,
            )

        except InvalidRegionError:
            logger.warning("Invalid region specified", region=request.region)
            return summoner_service_pb2.StartTrackingSummonerResponse(
                success=False,
                error_message=f"Invalid region: {request.region}",
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
            summoner_name=request.summoner_name,
            region=request.region,
        )

        try:
            # Use database session for repository operations
            async with self.db_manager.get_session() as session:
                tracked_player_repo = TrackedPlayerRepository(session)
                
                # Find the tracked player
                tracked_player = await tracked_player_repo.get_by_summoner_and_region(
                    request.summoner_name, request.region
                )

                if not tracked_player or not tracked_player.is_active:
                    logger.info(
                        "Summoner not currently being tracked",
                        summoner_name=request.summoner_name,
                        region=request.region,
                    )
                    return summoner_service_pb2.StopTrackingSummonerResponse(
                        success=False,
                        error_message=f"Summoner {request.summoner_name} is not currently being tracked in {request.region}",
                        error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_NOT_TRACKED,
                    )

                # Deactivate the tracked player
                await tracked_player_repo.set_active_status(tracked_player.id, False)
                await session.commit()

            logger.info(
                "Successfully stopped tracking summoner",
                summoner_name=request.summoner_name,
                region=request.region,
            )

            return summoner_service_pb2.StopTrackingSummonerResponse(success=True)

        except Exception as e:
            logger.error("Internal error during summoner untracking", error=str(e))
            return summoner_service_pb2.StopTrackingSummonerResponse(
                success=False,
                error_message="Internal service error. Please try again later.",
                error_code=summoner_service_pb2.ValidationError.VALIDATION_ERROR_INTERNAL_ERROR,
            )
