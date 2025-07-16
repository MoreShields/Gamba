"""Main service class for LoL Tracker."""

import asyncio
import logging
from typing import Optional
from concurrent.futures import ThreadPoolExecutor

import grpc
from grpc_reflection.v1alpha import reflection

from lol_tracker.config import Config
from lol_tracker.message_bus import MessageBusClient, NATSMessageBusClient
from lol_tracker.database.connection import DatabaseManager
from lol_tracker.summoner_service import SummonerTrackingService
from lol_tracker.proto.services import summoner_service_pb2_grpc, summoner_service_pb2


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
            # TODO: Start polling loop

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
