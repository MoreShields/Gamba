"""Main service class for LoL Tracker."""

import asyncio
import logging
from typing import Optional
from concurrent.futures import ThreadPoolExecutor

import grpc
from grpc_reflection.v1alpha import reflection

from lol_tracker.config import Config
from lol_tracker.adapters.messaging import NATSMessageBusClient, MessageBusClient
from lol_tracker.adapters.grpc.summoner_service import SummonerTrackingService
from lol_tracker.proto.services import summoner_service_pb2_grpc, summoner_service_pb2
from lol_tracker.adapters.database.manager import DatabaseManager
from lol_tracker.adapters.riot_api.client import RiotAPIClient
from lol_tracker.application.polling_service import PollingService
from lol_tracker.application.game_centric_polling_service import GameCentricPollingService
from lol_tracker.adapters.messaging.events import EventPublisher
from lol_tracker.adapters.observability import initialize_metrics, shutdown_metrics


logger = logging.getLogger(__name__)


class LoLTrackerService:
    """Main service class that orchestrates the LoL tracking functionality.
    
    This simplified service directly manages infrastructure components
    without complex dependency injection frameworks.
    """

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

        # Infrastructure components
        self._database_manager: Optional[DatabaseManager] = None
        self._message_bus_client: Optional[MessageBusClient] = None
        self._riot_api_client: Optional[RiotAPIClient] = None
        self._event_publisher: Optional[EventPublisher] = None
        self._polling_service: Optional[PollingService] = None
        self._metrics_provider = None
        
        # Provided dependencies
        self._provided_message_bus_client = message_bus_client

        # gRPC server and service
        self.grpc_server = None
        self.summoner_service = None

    async def _start_grpc_server(self):
        """Start the gRPC server."""
        logger.info("Starting gRPC server...")

        # Create gRPC server
        self.grpc_server = grpc.aio.server(
            ThreadPoolExecutor(max_workers=self.config.grpc_server_max_workers)
        )

        # Create summoner service with direct dependencies
        self.summoner_service = SummonerTrackingService(
            self._database_manager, 
            self._riot_api_client
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
            # Initialize infrastructure components directly
            await self._initialize_infrastructure()
            
            # Start gRPC server
            await self._start_grpc_server()

            # Start game state polling
            logger.info("Starting game state polling...")
            await self._polling_service.start_polling()

            # Main service loop - handles health checks and coordination
            while self._running:
                # Check message bus connection health
                if not await self._message_bus_client.is_connected():
                    logger.warning(
                        "Message bus connection lost, attempting to reconnect..."
                    )
                    try:
                        await self._message_bus_client.connect()
                        await self._message_bus_client.create_streams()
                        logger.info("Message bus reconnection successful")
                    except Exception as e:
                        logger.error(f"Failed to reconnect to message bus: {e}")

                # Health check interval
                await asyncio.sleep(min(self.config.poll_interval_seconds, 30))

        except Exception:
            self._running = False
            raise


    async def stop(self):
        """Stop the LoL Tracker service."""
        logger.info("Stopping LoL Tracker service")
        self._running = False

        # Stop game state polling service
        if self._polling_service:
            try:
                await self._polling_service.stop_polling()
            except Exception as e:
                logger.error(f"Error stopping polling service: {e}")

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

        # Clean up infrastructure components
        await self._cleanup_infrastructure()

        logger.info("LoL Tracker service stopped")

    async def _initialize_infrastructure(self) -> None:
        """Initialize all infrastructure components."""
        logger.info("Initializing infrastructure components")
        
        # Initialize metrics provider first
        self._metrics_provider = initialize_metrics(self.config)
        
        # Initialize database
        self._database_manager = DatabaseManager(self.config)
        await self._database_manager.initialize()
        
        # Initialize message bus
        if self._provided_message_bus_client is not None:
            self._message_bus_client = self._provided_message_bus_client
        else:
            self._message_bus_client = NATSMessageBusClient(
                servers=self.config.message_bus_url,
                timeout=self.config.message_bus_timeout_seconds,
                max_reconnect_attempts=self.config.message_bus_max_reconnect_attempts,
                reconnect_delay=self.config.message_bus_reconnect_delay_seconds,
                lol_events_stream=self.config.lol_events_stream,
                tracking_events_stream=self.config.tracking_events_stream,
                lol_events_subject=self.config.game_state_events_subject,
                tracking_events_subject=self.config.tracking_events_subject,
            )
        
        await self._message_bus_client.connect()
        
        # Verify message bus connection
        if not await self._message_bus_client.is_connected():
            raise RuntimeError("Failed to connect to message bus")
        
        # Create JetStream streams
        await self._message_bus_client.create_streams()
        
        # Initialize Riot API client
        self._riot_api_client = RiotAPIClient(
            self.config.riot_api_key,
            self.config.tft_riot_api_key,
            metrics=self._metrics_provider,
            base_url=self.config.riot_api_url,
            request_timeout=self.config.riot_api_timeout_seconds,
        )
        
        logger.info(f"Using Riot API at: {self.config.riot_api_url}")
        
        # Initialize event publisher
        self._event_publisher = EventPublisher(self.config, self._metrics_provider)
        await self._event_publisher.initialize()
        
        # Create polling service based on feature flag
        if self.config.use_game_centric_model:
            logger.info("Using game-centric polling model")
            self._polling_service = GameCentricPollingService(
                database=self._database_manager,
                riot_api=self._riot_api_client,
                event_publisher=self._event_publisher,
                metrics=self._metrics_provider,
                config=self.config
            )
        else:
            logger.info("Using legacy state-transition polling model")
            self._polling_service = PollingService(
                database=self._database_manager,
                riot_api=self._riot_api_client,
                event_publisher=self._event_publisher,
                config=self.config
            )
        
        logger.info("Infrastructure initialization completed")

    async def _cleanup_infrastructure(self) -> None:
        """Clean up all infrastructure components."""
        logger.info("Cleaning up infrastructure components")
        
        # Close Riot API client
        if self._riot_api_client:
            await self._riot_api_client.close()
        
        # Close event publisher
        if self._event_publisher:
            try:
                await self._event_publisher.close()
            except Exception as e:
                logger.error(f"Error closing event publisher: {e}")
        
        # Close message bus connection
        if self._message_bus_client:
            try:
                await self._message_bus_client.disconnect()
            except Exception as e:
                logger.error(f"Error during message bus disconnect: {e}")
        
        # Close database connection
        if self._database_manager:
            try:
                await self._database_manager.close()
            except Exception as e:
                logger.error(f"Error during database disconnect: {e}")
        
        # Shutdown metrics provider
        shutdown_metrics()
        
        logger.info("Infrastructure cleanup completed")
