"""Pytest configuration and fixtures for integration testing."""

import asyncio
import os
from typing import AsyncGenerator

import pytest
import pytest_asyncio
from sqlalchemy.ext.asyncio import AsyncSession
from testcontainers.postgres import PostgresContainer

from lol_tracker.config import Config, Environment
from lol_tracker.database.connection import DatabaseManager
from lol_tracker.database.models import Base


@pytest.fixture(scope="session")
def event_loop():
    """Create an instance of the default event loop for the test session."""
    loop = asyncio.new_event_loop()
    yield loop
    loop.close()


@pytest.fixture(scope="session")
def postgres_container():
    """Start a PostgreSQL container for testing."""
    with PostgresContainer("postgres:16-alpine", driver="psycopg2") as postgres:
        # Set environment variables for the container
        os.environ["TEST_DATABASE_URL"] = postgres.get_connection_url().replace(
            "psycopg2", "asyncpg"
        )
        yield postgres


@pytest_asyncio.fixture(scope="session")
async def test_config(postgres_container):
    """Create a test configuration with the container database URL."""
    database_url = os.environ["TEST_DATABASE_URL"]

    config = Config(
        database_url=database_url,
        database_name="test",  # Use the database created by testcontainer
        riot_api_key="test_api_key",
        environment=Environment.CI,
        log_level="DEBUG",
    )

    return config


@pytest_asyncio.fixture
async def db_manager(test_config: Config):
    """Create and initialize a database manager for testing (function-scoped for isolation)."""
    manager = DatabaseManager(test_config)
    await manager.initialize()

    # Create tables for each test
    await manager.create_tables()

    yield manager

    # Cleanup - drop tables and close
    try:
        await manager.drop_tables()
    except Exception:
        # Tables might not exist, ignore
        pass
    await manager.close()


@pytest_asyncio.fixture
async def db_session(db_manager: DatabaseManager) -> AsyncGenerator[AsyncSession, None]:
    """Create a database session for each test with automatic rollback."""
    async with db_manager.get_session() as session:
        # Start a transaction
        transaction = await session.begin()

        try:
            yield session
        finally:
            # Always rollback to ensure test isolation
            await transaction.rollback()


@pytest_asyncio.fixture
async def clean_db_session(
    db_manager: DatabaseManager,
) -> AsyncGenerator[AsyncSession, None]:
    """Create a database session that commits changes (for testing commit behavior)."""
    async with db_manager.get_session() as session:
        yield session
        # This session will commit changes automatically


@pytest_asyncio.fixture
async def lol_tracker_service(test_config: Config, db_manager: DatabaseManager):
    """Create LoLTrackerService with real database, mocked externals for testing."""
    from unittest.mock import AsyncMock
    from lol_tracker.service import LoLTrackerService
    
    # Create service with mocked message bus but real database
    mock_message_bus = AsyncMock()
    service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
    service.db_manager = db_manager
    
    # Initialize the service database manager
    await service.db_manager.initialize()
    
    yield service
    
    # Cleanup
    if hasattr(service, 'riot_api_client'):
        await service.riot_api_client.close()
