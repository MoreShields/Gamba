"""Main service class for LoL Tracker."""

import asyncio
import logging
import time
from typing import Optional
from concurrent.futures import ThreadPoolExecutor
from datetime import datetime

import grpc
from grpc_reflection.v1alpha import reflection
from google.protobuf.timestamp_pb2 import Timestamp

from lol_tracker.config import Config
from lol_tracker.message_bus import MessageBusClient, NATSMessageBusClient
from lol_tracker.database.connection import DatabaseManager
from lol_tracker.database.repository import TrackedPlayerRepository, GameStateRepository
from lol_tracker.riot_api_client import (
    RiotAPIClient,
    PlayerNotInGameError,
    CurrentGameInfo,
)
from lol_tracker.summoner_service import SummonerTrackingService
from lol_tracker.proto.services import summoner_service_pb2_grpc, summoner_service_pb2
from lol_tracker.proto.events import lol_events_pb2


logger = logging.getLogger(__name__)


class LoLTrackerService:
    """Main service class that orchestrates the LoL tracking functionality."""

    def __init__(
        self, config: Config, message_bus_client: Optional[MessageBusClient] = None
    ):
        """Initialize the LoL Tracker service.

        Args:
            config: Service configuration
            message_bus_client: Optional message bus client for dependency injection.
                               If not provided, will create NATSMessageBusClient from config.
        """
        self.config = config
        self._running = False
        self._tasks: list[asyncio.Task] = []

        # Initialize message bus client with dependency injection support
        if message_bus_client is None:
            self.message_bus = NATSMessageBusClient(
                servers=config.message_bus_url,
                timeout=config.message_bus_timeout_seconds,
                max_reconnect_attempts=config.message_bus_max_reconnect_attempts,
                reconnect_delay=config.message_bus_reconnect_delay_seconds,
                lol_events_stream=config.lol_events_stream,
                tracking_events_stream=config.tracking_events_stream,
                lol_events_subject=config.game_state_events_subject,
                tracking_events_subject=config.tracking_events_subject,
            )
        else:
            self.message_bus = message_bus_client

        # Initialize database manager
        self.db_manager = DatabaseManager(config)

        # Initialize Riot API client
        self.riot_api_client = RiotAPIClient(
            api_key=config.riot_api_key, request_timeout=config.riot_api_timeout_seconds
        )

        # Initialize gRPC server and service
        self.grpc_server = None
        self.summoner_service = None

    async def _start_grpc_server(self):
        """Start the gRPC server."""
        logger.info("Starting gRPC server...")

        # Create gRPC server
        self.grpc_server = grpc.aio.server(
            ThreadPoolExecutor(max_workers=self.config.grpc_server_max_workers)
        )

        # Create summoner service
        self.summoner_service = SummonerTrackingService(
            self.db_manager, self.config.riot_api_key
        )

        # Add service to server
        summoner_service_pb2_grpc.add_SummonerTrackingServiceServicer_to_server(
            self.summoner_service, self.grpc_server
        )

        # Add reflection if enabled
        if self.config.grpc_server_reflection:
            SERVICE_NAMES = (
                summoner_service_pb2.DESCRIPTOR.services_by_name[
                    "SummonerTrackingService"
                ].full_name,
                reflection.SERVICE_NAME,
            )
            reflection.enable_server_reflection(SERVICE_NAMES, self.grpc_server)

        # Start server
        listen_addr = f"[::]:{self.config.grpc_server_port}"
        self.grpc_server.add_insecure_port(listen_addr)
        await self.grpc_server.start()

        logger.info(f"gRPC server started on port {self.config.grpc_server_port}")

    async def start(self):
        """Start the LoL Tracker service."""
        logger.info("Starting LoL Tracker service")
        self._running = True

        try:
            # Initialize database connection
            logger.info("Initializing database connection...")
            await self.db_manager.initialize()

            # Initialize message bus connection
            logger.info("Connecting to message bus...")
            await self.message_bus.connect()

            # Verify connection
            if not await self.message_bus.is_connected():
                raise RuntimeError("Failed to connect to message bus")

            # Create JetStream streams
            logger.info("Creating JetStream streams...")
            await self.message_bus.create_streams()

            logger.info("Message bus initialization completed successfully")

            # Start gRPC server
            await self._start_grpc_server()

            # TODO: Set up message subscriptions

            # Start polling loop
            logger.info("Starting game state polling task...")
            polling_task = asyncio.create_task(self._polling_loop())
            self._tasks.append(polling_task)

            # For now, just run a simple loop to keep service alive
            while self._running:
                logger.info("LoL Tracker service running...")

                # Check message bus connection health
                if not await self.message_bus.is_connected():
                    logger.warning(
                        "Message bus connection lost, attempting to reconnect..."
                    )
                    try:
                        await self.message_bus.connect()
                        await self.message_bus.create_streams()
                        logger.info("Message bus reconnection successful")
                    except Exception as e:
                        logger.error(f"Failed to reconnect to message bus: {e}")

                await asyncio.sleep(self.config.poll_interval_seconds)

        except Exception as e:
            logger.error(f"Failed to start LoL Tracker service: {e}")
            self._running = False
            raise

    async def _polling_loop(self):
        """Main polling loop that monitors tracked players' game states."""
        logger.info("Game state polling loop started")

        while self._running:
            try:
                # Get all active tracked players
                async with self.db_manager.get_session() as session:
                    # Create repository instance with session
                    tracked_player_repo = TrackedPlayerRepository(session)
                    tracked_players = await tracked_player_repo.get_all_active()

                if not tracked_players:
                    logger.debug("No tracked players found, skipping poll cycle")
                    await asyncio.sleep(self.config.poll_interval_seconds)
                    continue

                logger.info(
                    f"Polling game state for {len(tracked_players)} tracked players"
                )

                # Poll each tracked player
                for player in tracked_players:
                    if not self._running:
                        break

                    try:
                        await self._poll_player_game_state(player)
                    except Exception as e:
                        logger.error(
                            f"Error polling player {player.game_name}#{player.tag_line}: {e}"
                        )

                # Wait before next poll cycle
                await asyncio.sleep(self.config.poll_interval_seconds)

            except Exception as e:
                logger.error(f"Error in polling loop: {e}")
                # Wait before retrying on error
                await asyncio.sleep(min(self.config.poll_interval_seconds, 30))

        logger.info("Game state polling loop stopped")

    async def _poll_player_game_state(self, player):
        """Poll a single player's game state and detect changes."""
        if not player.puuid:
            logger.warning(
                f"Player {player.game_name}#{player.tag_line} has no puuid, skipping"
            )
            return

        async with self.db_manager.get_session() as session:
            # Create repository instance with session
            game_state_repo = GameStateRepository(session)
            # Get current game state from database
            current_db_state = await game_state_repo.get_latest_for_player(player.id)

            # Get current game state from Riot API
            riot_game_state = await self._get_riot_game_state(player)

            # Determine if state changed
            previous_status = self._get_game_status_from_db_state(current_db_state)
            current_status = self._get_game_status_from_riot_state(riot_game_state)

            if previous_status != current_status:
                logger.info(
                    f"Game state changed for {player.game_name}#{player.tag_line}: "
                    f"{lol_events_pb2.GameStatus.Name(previous_status)} -> {lol_events_pb2.GameStatus.Name(current_status)}"
                )

                # Create new game state record
                new_game_state = await self._create_game_state_record(
                    session, player, riot_game_state, current_status
                )

                # Emit event for state change
                await self._emit_game_state_changed_event(
                    player,
                    previous_status,
                    current_status,
                    new_game_state,
                    riot_game_state,
                )
            else:
                # Update existing state if in game (for game length tracking)
                if (
                    current_status == lol_events_pb2.GameStatus.GAME_STATUS_IN_GAME
                    and current_db_state
                ):
                    await self._update_in_game_state(
                        session, current_db_state, riot_game_state
                    )

    async def _get_riot_game_state(self, player) -> Optional[CurrentGameInfo]:
        """Get current game state from Riot API."""
        try:
            return await self.riot_api_client.get_current_game_info(
                player.puuid, player.game_name, "na1"
            )
        except PlayerNotInGameError:
            return None
        except Exception as e:
            logger.error(
                f"Error fetching game state for {player.game_name}#{player.tag_line}: {e}"
            )
            return None

    def _get_game_status_from_db_state(self, db_state) -> lol_events_pb2.GameStatus:
        """Convert database game state to protobuf GameStatus."""
        if not db_state:
            return lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME

        status_map = {
            "NOT_IN_GAME": lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME,
            "IN_CHAMPION_SELECT": lol_events_pb2.GameStatus.GAME_STATUS_IN_CHAMPION_SELECT,
            "IN_GAME": lol_events_pb2.GameStatus.GAME_STATUS_IN_GAME,
        }
        return status_map.get(
            db_state.status, lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME
        )

    def _get_game_status_from_riot_state(
        self, riot_state: Optional[CurrentGameInfo]
    ) -> lol_events_pb2.GameStatus:
        """Convert Riot API game state to protobuf GameStatus."""
        if not riot_state:
            return lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME

        # For now, we treat all active games as IN_GAME
        # Champion select detection would require additional API calls
        return lol_events_pb2.GameStatus.GAME_STATUS_IN_GAME

    async def _create_game_state_record(self, session, player, riot_state, status):
        """Create a new game state record in the database."""
        if riot_state:
            # Player is in game
            game_state_data = {
                "player_id": player.id,
                "status": "IN_GAME",
                "game_id": riot_state.game_id,
                "queue_type": riot_state.queue_type,
                "game_start_time": datetime.fromtimestamp(
                    riot_state.game_start_time / 1000
                ),
                "raw_api_response": str(riot_state.__dict__),
            }
        else:
            # Player is not in game
            game_state_data = {
                "player_id": player.id,
                "status": "NOT_IN_GAME",
            }

        game_state_repo = GameStateRepository(session)
        return await game_state_repo.create(**game_state_data)

    async def _update_in_game_state(self, session, db_state, riot_state):
        """Update an existing in-game state with current info."""
        if riot_state and db_state.game_id == riot_state.game_id:
            # Update game length and other live data
            updates = {
                "raw_api_response": str(riot_state.__dict__),
            }
            # Note: Could update other fields like game_length here if needed
            # For now, we keep it minimal

    async def _emit_game_state_changed_event(
        self, player, previous_status, current_status, game_state, riot_state
    ):
        """Emit a LoLGameStateChanged event to the message bus."""
        try:
            # Create timestamp
            timestamp = Timestamp()
            timestamp.GetCurrentTime()

            # Create the event
            event = lol_events_pb2.LoLGameStateChanged(
                game_name=player.game_name,
                tag_line=player.tag_line,
                previous_status=previous_status,
                current_status=current_status,
                event_time=timestamp,
            )

            # Add game metadata if available
            if riot_state:
                event.game_id = riot_state.game_id
                event.queue_type = riot_state.queue_type

            # TODO: Add game result when transitioning out of IN_GAME
            # This would require fetching match details from the Match API

            # Publish the event
            subject = "lol.gamestate.changed"
            await self.message_bus.publish(subject, event.SerializeToString())

            logger.info(
                f"Published game state changed event for {player.game_name}#{player.tag_line}: "
                f"{lol_events_pb2.GameStatus.Name(previous_status)} -> {lol_events_pb2.GameStatus.Name(current_status)}"
            )

        except Exception as e:
            logger.error(
                f"Error publishing game state changed event for {player.game_name}#{player.tag_line}: {e}"
            )

    async def stop(self):
        """Stop the LoL Tracker service."""
        logger.info("Stopping LoL Tracker service")
        self._running = False

        # Cancel all running tasks
        for task in self._tasks:
            task.cancel()

        # Wait for tasks to complete
        if self._tasks:
            await asyncio.gather(*self._tasks, return_exceptions=True)

        # Stop gRPC server
        if self.grpc_server:
            logger.info("Stopping gRPC server...")
            await self.grpc_server.stop(grace=30)
            logger.info("gRPC server stopped")

        # Close summoner service
        if self.summoner_service:
            await self.summoner_service.close()

        # Close Riot API client
        await self.riot_api_client.close()

        # Close message bus connection
        try:
            logger.info("Disconnecting from message bus...")
            await self.message_bus.disconnect()
            logger.info("Message bus disconnection completed")
        except Exception as e:
            logger.error(f"Error during message bus disconnect: {e}")

        # Close database connection
        try:
            logger.info("Closing database connection...")
            await self.db_manager.close()
            logger.info("Database connection closed")
        except Exception as e:
            logger.error(f"Error during database disconnect: {e}")

        logger.info("LoL Tracker service stopped")
