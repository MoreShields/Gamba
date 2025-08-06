"""Simplified pytest fixtures for LoL Tracker integration tests."""

import asyncio
import os
import subprocess
import sys
from pathlib import Path
import pytest
import pytest_asyncio
from testcontainers.postgres import PostgresContainer
from aiohttp import web
from unittest.mock import AsyncMock

# Add the parent directory to the path if not already there
# This ensures the lol_tracker module can be imported in CI
project_root = Path(__file__).parent.parent
if str(project_root) not in sys.path:
    sys.path.insert(0, str(project_root))

from lol_tracker.config import Config
from lol_tracker.service import LoLTrackerService
from lol_tracker.adapters.database.manager import DatabaseManager
from lol_tracker.adapters.riot_api.client import RiotAPIClient
from lol_tracker.adapters.messaging.events import MockEventPublisher
from lol_tracker.application.polling_service import PollingService
from lol_tracker.proto.services import summoner_service_pb2_grpc, summoner_service_pb2
from lol_tracker.proto.events import lol_events_pb2, tft_events_pb2
from mock_riot_api.mock_riot_server import MockRiotAPIServer
from mock_riot_api.control import MockRiotControlClient
import grpc
import asyncio
from sqlalchemy import text
from lol_tracker.proto.services import summoner_service_pb2


@pytest.fixture(scope="session")
def postgres_container():
    """Create a PostgreSQL container for testing."""
    container = PostgresContainer("postgres:14-alpine")
    container.start()
    yield container
    container.stop()


@pytest.fixture
def test_config(postgres_container):
    """Create test configuration."""
    # Convert psycopg2 URL to asyncpg
    sync_url = postgres_container.get_connection_url()
    database_url = sync_url.replace("postgresql+psycopg2", "postgresql+asyncpg")
    
    # Set environment for Config.from_env()
    os.environ["DATABASE_URL"] = database_url
    os.environ["DATABASE_NAME"] = "test"
    os.environ["RIOT_API_KEY"] = "test-api-key"
    os.environ["RIOT_API_URL"] = "http://localhost:8081"
    os.environ["POLL_INTERVAL_SECONDS"] = "1"
    os.environ["MESSAGE_BUS_URL"] = "nats://localhost:4222"
    os.environ["ENVIRONMENT"] = "CI"
    os.environ["GRPC_SERVER_PORT"] = "50052"
    
    return Config.from_env()


@pytest_asyncio.fixture
async def mock_riot_api_server():
    """Start mock Riot API server."""
    server = MockRiotAPIServer(port=8081)
    
    # Start server in background
    runner = web.AppRunner(server.app)
    await runner.setup()
    site = web.TCPSite(runner, 'localhost', 8081)
    await site.start()
    
    # Wait for server to be ready
    await asyncio.sleep(0.5)
    
    yield server
    
    await runner.cleanup()


@pytest_asyncio.fixture
async def database_manager(test_config):
    """Initialize database with migrations."""
    # Run alembic migrations
    env = os.environ.copy()
    env["DATABASE_URL"] = test_config.get_database_url()
    
    # Use sys.executable to get the current Python interpreter
    # project_root is already defined at the top of the file
    
    result = subprocess.run(
        [sys.executable, "-m", "alembic", "upgrade", "head"],
        cwd=str(project_root),
        env=env,
        capture_output=True,
        text=True,
        timeout=10
    )
    
    if result.returncode != 0:
        raise RuntimeError(f"Migration failed: {result.stderr}")
    
    # Create and initialize manager
    manager = DatabaseManager(test_config)
    await manager.initialize()
    
    yield manager
    
    # No need to run downgrade - testcontainer will be cleaned up automatically
    await manager.close()


@pytest_asyncio.fixture
async def mock_event_publisher(test_config):
    """Create mock event publisher."""
    publisher = MockEventPublisher(test_config)
    await publisher.initialize()
    yield publisher
    await publisher.close()


@pytest_asyncio.fixture
async def lol_tracker_service(test_config, database_manager, mock_event_publisher, mock_riot_api_server):
    """Create minimal LoL Tracker service for testing."""
    # Mock NATS to avoid connection issues
    mock_nats = AsyncMock()
    mock_nats.is_connected.return_value = True
    
    # Create service with mocked NATS
    service = LoLTrackerService(test_config, message_bus_client=mock_nats)
    
    # Manually wire dependencies
    service._database_manager = database_manager
    service._event_publisher = mock_event_publisher
    service._riot_api_client = RiotAPIClient(
        api_key=test_config.riot_api_key,
        base_url=test_config.riot_api_url,
        request_timeout=test_config.riot_api_timeout_seconds
    )
    service._message_bus_client = mock_nats
    
    # Create polling service
    service._polling_service = PollingService(
        database=database_manager,
        riot_api=service._riot_api_client,
        event_publisher=mock_event_publisher,
        config=test_config
    )
    
    # Start only gRPC server
    service._running = True
    await service._start_grpc_server()
    
    # Start polling in background (runs automatically every 1 second)
    await service._polling_service.start_polling()
    
    yield service
    
    # Cleanup
    service._running = False
    await service._polling_service.stop_polling()
    if service.grpc_server:
        await service.grpc_server.stop(grace=1)
    await service._riot_api_client.close()


@pytest_asyncio.fixture
async def grpc_client(test_config):
    """Create gRPC client."""
    channel = grpc.aio.insecure_channel(f'localhost:{test_config.grpc_server_port}')
    stub = summoner_service_pb2_grpc.SummonerTrackingServiceStub(channel)
    yield stub
    await channel.close()


@pytest_asyncio.fixture
async def mock_riot_control():
    """Create mock Riot API control client."""
    client = MockRiotControlClient("http://localhost:8081")
    await client.reset_server()
    return client


class BaseE2ETest:
    """Base class for E2E tracker tests with common utilities."""
    
    async def setup_test_environment(self, mock_riot_control, mock_event_publisher, database_manager):
        """Clean slate setup for each test."""
        await self._cleanup_database(database_manager)
        await mock_riot_control.reset_server()
        mock_event_publisher.published_messages.clear()
    
    async def _cleanup_database(self, database_manager):
        """Clean up database from previous tests."""
        async with database_manager.get_session() as session:
            await session.execute(text("DELETE FROM game_states"))
            await session.execute(text("DELETE FROM tracked_players"))
            await session.commit()
    
    async def create_test_player(self, mock_riot_control, game_name: str, tag_line: str, puuid: str):
        """Create a test player in the mock Riot API."""
        player_data = await mock_riot_control.create_player(game_name, tag_line, puuid)
        assert player_data["puuid"] == puuid
        return player_data
    
    async def track_summoner_via_grpc(self, grpc_client, game_name: str, tag_line: str, expected_puuid: str):
        """Track a summoner via gRPC and verify the response."""
        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name=game_name,
            tag_line=tag_line
        )
        response = await grpc_client.StartTrackingSummoner(request)
        assert response.success is True
        assert response.summoner_details.puuid == expected_puuid
        return response
    
    async def verify_player_tracked_in_db(self, database_manager, puuid: str, game_name: str, tag_line: str):
        """Verify player is properly tracked in the database."""
        tracked_player = await database_manager.get_tracked_player_by_puuid(puuid)
        assert tracked_player is not None
        assert tracked_player.game_name == game_name
        assert tracked_player.tag_line == tag_line
        return tracked_player
    
    async def wait_for_polling_cycle(self, wait_time: float = 2.0):
        """Wait for automatic polling to detect changes."""
        await asyncio.sleep(wait_time)
    
    def find_events_by_type(self, mock_event_publisher, event_filter):
        """Find events matching a given filter function."""
        return [e for e in mock_event_publisher.published_messages if event_filter(e)]
    
    def find_game_state_events(self, mock_event_publisher):
        """Find all game state change events."""
        return self.find_events_by_type(
            mock_event_publisher, 
            lambda e: "state_changed" in e.get("subject", "")
        )
    
    def find_game_end_events(self, mock_event_publisher):
        """Find all game end events."""
        # With protobuf messages, we need to check the actual message content
        game_end_events = []
        for msg in mock_event_publisher.published_messages:
            pb_msg = msg["protobuf_message"]
            # Check if this is a game end event by looking at status transition
            if hasattr(pb_msg, 'previous_status') and hasattr(pb_msg, 'current_status'):
                # For LoL events
                if msg["message_type"] == "LoLGameStateChanged":
                    if (pb_msg.previous_status == lol_events_pb2.GAME_STATUS_IN_GAME and 
                        pb_msg.current_status == lol_events_pb2.GAME_STATUS_NOT_IN_GAME):
                        game_end_events.append(msg)
                # For TFT events
                elif msg["message_type"] == "TFTGameStateChanged":
                    if (pb_msg.previous_status == tft_events_pb2.TFT_GAME_STATUS_IN_GAME and 
                        pb_msg.current_status == tft_events_pb2.TFT_GAME_STATUS_NOT_IN_GAME):
                        game_end_events.append(msg)
        return game_end_events
    
    def assert_game_start_event(self, event, game_id: str):
        """Assert that an event represents a proper game start."""
        pb_msg = event["protobuf_message"]
        
        # Check status transition based on message type
        if event["message_type"] == "LoLGameStateChanged":
            assert pb_msg.previous_status == lol_events_pb2.GAME_STATUS_NOT_IN_GAME
            assert pb_msg.current_status == lol_events_pb2.GAME_STATUS_IN_GAME
        elif event["message_type"] == "TFTGameStateChanged":
            assert pb_msg.previous_status == tft_events_pb2.TFT_GAME_STATUS_NOT_IN_GAME
            assert pb_msg.current_status == tft_events_pb2.TFT_GAME_STATUS_IN_GAME
        
        assert pb_msg.game_id == game_id
    
    def assert_game_end_event(self, event, game_id: str = None):
        """Assert that an event represents a proper game end."""
        pb_msg = event["protobuf_message"]
        
        # Check status transition based on message type
        if event["message_type"] == "LoLGameStateChanged":
            assert pb_msg.previous_status == lol_events_pb2.GAME_STATUS_IN_GAME
            assert pb_msg.current_status == lol_events_pb2.GAME_STATUS_NOT_IN_GAME
        elif event["message_type"] == "TFTGameStateChanged":
            assert pb_msg.previous_status == tft_events_pb2.TFT_GAME_STATUS_IN_GAME
            assert pb_msg.current_status == tft_events_pb2.TFT_GAME_STATUS_NOT_IN_GAME
        
        if game_id:
            assert pb_msg.game_id == game_id
    
    def assert_game_result_present(self, event):
        """Assert that game result data is present in the event."""
        pb_msg = event["protobuf_message"]
        
        # Check if game_result is present and populated
        assert hasattr(pb_msg, 'game_result')
        assert pb_msg.HasField('game_result')
        
        # Return a dictionary representation of the game result for easier testing
        game_result = pb_msg.game_result
        result_dict = {}
        
        if event["message_type"] == "LoLGameStateChanged":
            result_dict["won"] = game_result.won
            result_dict["duration_seconds"] = game_result.duration_seconds
            result_dict["champion_played"] = game_result.champion_played
            if game_result.queue_type:
                result_dict["queue_type"] = game_result.queue_type
        elif event["message_type"] == "TFTGameStateChanged":
            result_dict["placement"] = game_result.placement
            result_dict["duration_seconds"] = game_result.duration_seconds
            result_dict["won"] = game_result.placement <= 4  # Top 4 is a win in TFT
        
        return result_dict
    
    async def verify_game_state_in_db(self, database_manager, tracked_player, expected_status: str, expected_game_id: str = None):
        """Verify game state is correctly stored in database."""
        game_state = await database_manager.get_latest_game_state_for_player(tracked_player.id)
        assert game_state is not None
        assert game_state.status.value == expected_status
        if expected_game_id:
            assert game_state.game_id == expected_game_id
        return game_state