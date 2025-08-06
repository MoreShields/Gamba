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
from lol_tracker.proto.services import summoner_service_pb2_grpc
from mock_riot_api.mock_riot_server import MockRiotAPIServer
from mock_riot_api.control import MockRiotControlClient
import grpc


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