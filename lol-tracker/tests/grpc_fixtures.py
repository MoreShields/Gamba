"""Shared gRPC test fixtures and infrastructure."""

import asyncio
import pytest
import pytest_asyncio
from concurrent.futures import ThreadPoolExecutor
from typing import AsyncGenerator

import grpc
from grpc_reflection.v1alpha import reflection

from lol_tracker.summoner_service import SummonerTrackingService
from lol_tracker.proto.services import summoner_service_pb2_grpc, summoner_service_pb2
from lol_tracker.database.connection import DatabaseManager
from lol_tracker.riot_api import RiotAPIClient
from lol_tracker.config import Config
from lol_tracker.riot_api import create_riot_api_client


@pytest_asyncio.fixture
async def riot_api_client(test_config: Config) -> RiotAPIClient:
    """Create a Riot API client for testing (uses mock if MOCK_RIOT_API_URL is set)."""
    client = create_riot_api_client(test_config)
    yield client
    await client.close()


@pytest_asyncio.fixture
async def summoner_service(db_manager: DatabaseManager, riot_api_client: RiotAPIClient) -> SummonerTrackingService:
    """Create a SummonerTrackingService instance for testing."""
    service = SummonerTrackingService(db_manager, riot_api_client)
    yield service
    # No need to close service anymore as it doesn't own the riot_api_client


@pytest_asyncio.fixture
async def grpc_server(
    summoner_service: SummonerTrackingService,
) -> AsyncGenerator[grpc.aio.Server, None]:
    """Create and start a gRPC server for testing."""
    # Create gRPC server
    server = grpc.aio.server(ThreadPoolExecutor(max_workers=10))

    # Add service to server
    summoner_service_pb2_grpc.add_SummonerTrackingServiceServicer_to_server(
        summoner_service, server
    )

    # Add reflection for debugging
    SERVICE_NAMES = (
        summoner_service_pb2.DESCRIPTOR.services_by_name[
            "SummonerTrackingService"
        ].full_name,
        reflection.SERVICE_NAME,
    )
    reflection.enable_server_reflection(SERVICE_NAMES, server)

    # Find available port and start server
    port = server.add_insecure_port("[::]:0")  # 0 means assign any available port
    await server.start()

    # Store the port for client connections
    server._test_port = port

    yield server

    # Cleanup
    await server.stop(grace=5)


@pytest_asyncio.fixture
async def grpc_channel(
    grpc_server: grpc.aio.Server,
) -> AsyncGenerator[grpc.aio.Channel, None]:
    """Create a gRPC channel connected to the test server."""
    port = grpc_server._test_port
    channel = grpc.aio.insecure_channel(f"localhost:{port}")

    # Wait for channel to be ready
    await channel.channel_ready()

    yield channel

    # Cleanup
    await channel.close()


@pytest_asyncio.fixture
async def grpc_client(
    grpc_channel: grpc.aio.Channel,
) -> summoner_service_pb2_grpc.SummonerTrackingServiceStub:
    """Create a gRPC client stub for making requests."""
    return summoner_service_pb2_grpc.SummonerTrackingServiceStub(grpc_channel)


@pytest.fixture
def sample_start_tracking_request() -> (
    summoner_service_pb2.StartTrackingSummonerRequest
):
    """Create a sample StartTrackingSummonerRequest for testing."""
    from google.protobuf.timestamp_pb2 import Timestamp

    timestamp = Timestamp()
    timestamp.GetCurrentTime()

    return summoner_service_pb2.StartTrackingSummonerRequest(
        game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
    )


@pytest.fixture
def sample_stop_tracking_request() -> summoner_service_pb2.StopTrackingSummonerRequest:
    """Create a sample StopTrackingSummonerRequest for testing."""
    from google.protobuf.timestamp_pb2 import Timestamp

    timestamp = Timestamp()
    timestamp.GetCurrentTime()

    return summoner_service_pb2.StopTrackingSummonerRequest(
        game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
    )
