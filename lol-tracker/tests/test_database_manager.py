"""Integration tests for DatabaseManager."""

import pytest
import pytest_asyncio
from unittest.mock import patch
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy import text

from lol_tracker.config import Config, Environment
from lol_tracker.database.connection import (
    DatabaseManager,
    initialize_database,
    close_database,
    get_database_manager,
)
from lol_tracker.database.models import TrackedPlayer, GameState
from lol_tracker.database.repository import TrackedPlayerRepository, GameStateRepository


@pytest.mark.integration
class TestDatabaseManager:
    """Test suite for DatabaseManager integration tests."""

    @pytest.mark.asyncio
    async def test_database_manager_initialization(self, test_config: Config):
        """Test database manager initialization and cleanup."""
        manager = DatabaseManager(test_config)

        # Initially not initialized
        with pytest.raises(RuntimeError, match="not initialized"):
            async with manager.get_session():
                pass

        # Initialize
        await manager.initialize()

        # Should work after initialization
        async with manager.get_session() as session:
            assert isinstance(session, AsyncSession)

        # Cleanup
        await manager.close()

        # Should not work after cleanup
        with pytest.raises(RuntimeError, match="not initialized"):
            async with manager.get_session():
                pass

    @pytest.mark.asyncio
    async def test_double_initialization_warning(self, test_config: Config, caplog):
        """Test that double initialization logs a warning."""
        manager = DatabaseManager(test_config)

        await manager.initialize()

        # Second initialization should warn
        await manager.initialize()

        assert "already initialized" in caplog.text

        await manager.close()

    @pytest.mark.asyncio
    async def test_session_context_manager(self, db_manager: DatabaseManager):
        """Test session context manager behavior."""
        async with db_manager.get_session() as session:
            # Session should be active
            assert session.is_active

            # Should be able to execute queries
            result = await session.execute(text("SELECT 1"))
            assert result.scalar() == 1

    @pytest.mark.asyncio
    async def test_session_rollback_on_exception(self, db_manager: DatabaseManager):
        """Test that session rolls back on exception."""
        player_repo = None

        with pytest.raises(ValueError):
            async with db_manager.get_session() as session:
                player_repo = TrackedPlayerRepository(session)

                # Create a player
                await player_repo.create("TestPlayer", "NA1")

                # Raise an exception to trigger rollback
                raise ValueError("Test exception")

        # Verify the player was not created due to rollback
        async with db_manager.get_session() as session:
            new_repo = TrackedPlayerRepository(session)
            player = await new_repo.get_by_summoner_and_region("TestPlayer", "NA1")
            assert player is None

    @pytest.mark.asyncio
    async def test_session_commit_behavior(self, db_manager: DatabaseManager):
        """Test that changes are committed when session completes successfully."""
        # Create a player in one session
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            player = await player_repo.create("CommitTest", "NA1")
            await session.commit()
            player_id = player.id

        # Verify the player exists in a new session
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            retrieved_player = await player_repo.get_by_id(player_id)
            assert retrieved_player is not None
            assert retrieved_player.summoner_name == "CommitTest"

    @pytest.mark.asyncio
    async def test_create_and_drop_tables(self, test_config: Config):
        """Test table creation and dropping."""
        manager = DatabaseManager(test_config)
        await manager.initialize()

        # Drop tables if they exist
        await manager.drop_tables()

        # Create tables
        await manager.create_tables()

        # Verify tables work by creating some data
        async with manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            player = await player_repo.create("TableTest", "NA1")
            await session.commit()
            assert player.id is not None

        # Drop tables
        await manager.drop_tables()

        await manager.close()

    @pytest.mark.asyncio
    async def test_create_tables_without_initialization(self, test_config: Config):
        """Test that create_tables fails without initialization."""
        manager = DatabaseManager(test_config)

        with pytest.raises(RuntimeError, match="not initialized"):
            await manager.create_tables()

    @pytest.mark.asyncio
    async def test_drop_tables_without_initialization(self, test_config: Config):
        """Test that drop_tables fails without initialization."""
        manager = DatabaseManager(test_config)

        with pytest.raises(RuntimeError, match="not initialized"):
            await manager.drop_tables()

    @pytest.mark.asyncio
    async def test_engine_property(self, db_manager: DatabaseManager):
        """Test the engine property."""
        engine = db_manager.engine
        assert engine is not None

        # Engine should be usable
        async with engine.begin() as conn:
            result = await conn.execute(text("SELECT 1"))
            assert result.scalar() == 1

    @pytest.mark.asyncio
    async def test_engine_property_without_initialization(self, test_config: Config):
        """Test that engine property fails without initialization."""
        manager = DatabaseManager(test_config)

        with pytest.raises(RuntimeError, match="not initialized"):
            _ = manager.engine

    @pytest.mark.asyncio
    async def test_concurrent_sessions(self, db_manager: DatabaseManager):
        """Test multiple concurrent sessions."""
        import asyncio

        async def create_player(name: str):
            async with db_manager.get_session() as session:
                player_repo = TrackedPlayerRepository(session)
                player = await player_repo.create(name, "NA1")
                await session.commit()
                return player.id

        # Create multiple players concurrently
        tasks = [create_player(f"Concurrent{i}") for i in range(5)]
        player_ids = await asyncio.gather(*tasks)

        # Verify all players were created
        assert len(player_ids) == 5
        assert len(set(player_ids)) == 5  # All unique IDs

        # Verify players exist
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            for player_id in player_ids:
                player = await player_repo.get_by_id(player_id)
                assert player is not None

    @pytest.mark.asyncio
    async def test_nested_transactions(self, db_manager: DatabaseManager):
        """Test nested transaction behavior."""
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)

            # Create a player
            player = await player_repo.create("NestedTest", "NA1")
            await session.flush()  # Flush but don't commit

            # Start a nested transaction (savepoint)
            savepoint = await session.begin_nested()

            try:
                # Create another player in the nested transaction
                await player_repo.create("NestedTest2", "NA1")
                await session.flush()

                # Rollback the nested transaction
                await savepoint.rollback()
            except Exception:
                await savepoint.rollback()
                raise

            # Commit the outer transaction
            await session.commit()

        # Verify only the first player exists
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            player1 = await player_repo.get_by_summoner_and_region("NestedTest", "NA1")
            player2 = await player_repo.get_by_summoner_and_region("NestedTest2", "NA1")

            assert player1 is not None
            assert player2 is None

    @pytest.mark.asyncio
    async def test_repository_integration(self, db_manager: DatabaseManager):
        """Test full integration with repository classes."""
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            game_repo = GameStateRepository(session)

            # Create a player
            player = await player_repo.create("IntegrationTest", "NA1")
            await session.flush()

            # Create game states for the player
            state1 = await game_repo.create(player.id, "IN_CHAMPION_SELECT")
            state2 = await game_repo.create(player.id, "IN_GAME")
            await session.flush()

            # Verify relationships work
            latest_state = await game_repo.get_latest_for_player(player.id)
            assert latest_state.id == state2.id

            # Test cascade delete
            await player_repo.delete(player.id)
            await session.commit()

        # Verify player and related game states are deleted
        async with db_manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            game_repo = GameStateRepository(session)

            deleted_player = await player_repo.get_by_summoner_and_region(
                "IntegrationTest", "NA1"
            )
            remaining_states = await game_repo.get_recent_games_for_player(player.id)

            assert deleted_player is None
            assert remaining_states == []


@pytest.mark.integration
class TestGlobalDatabaseManager:
    """Test suite for global database manager functions."""

    @pytest_asyncio.fixture(autouse=True)
    async def cleanup_global_manager(self):
        """Ensure global manager is cleaned up after each test."""
        yield
        await close_database()

    @pytest.mark.asyncio
    async def test_initialize_global_database(self, test_config: Config):
        """Test initializing the global database manager."""
        manager = await initialize_database(test_config)

        assert manager is not None

        # Should be able to get the manager
        retrieved_manager = get_database_manager()
        assert retrieved_manager is manager

        await close_database()

    @pytest.mark.asyncio
    async def test_double_initialize_global_database(self, test_config: Config, caplog):
        """Test double initialization of global database manager."""
        manager1 = await initialize_database(test_config)
        manager2 = await initialize_database(test_config)

        assert manager1 is manager2
        assert "already initialized" in caplog.text

        await close_database()

    @pytest.mark.asyncio
    async def test_get_database_manager_not_initialized(self):
        """Test getting database manager when not initialized."""
        with pytest.raises(RuntimeError, match="not initialized"):
            get_database_manager()

    @pytest.mark.asyncio
    async def test_close_global_database(self, test_config: Config):
        """Test closing the global database manager."""
        await initialize_database(test_config)

        # Should be able to get manager
        manager = get_database_manager()
        assert manager is not None

        # Close the database
        await close_database()

        # Should no longer be able to get manager
        with pytest.raises(RuntimeError, match="not initialized"):
            get_database_manager()

    @pytest.mark.asyncio
    async def test_close_global_database_not_initialized(self):
        """Test closing global database when not initialized."""
        # Should not raise an error
        await close_database()

    @pytest.mark.asyncio
    async def test_global_database_usage(self, test_config: Config):
        """Test using the global database manager for operations."""
        await initialize_database(test_config)
        manager = get_database_manager()

        # Create tables
        await manager.create_tables()

        # Use the global manager for database operations
        async with manager.get_session() as session:
            player_repo = TrackedPlayerRepository(session)
            player = await player_repo.create("GlobalTest", "NA1")
            await session.commit()
            assert player.id is not None

        await close_database()

    @pytest.mark.asyncio
    async def test_error_handling_in_global_context(self, test_config: Config):
        """Test error handling when using global database manager."""
        await initialize_database(test_config)
        manager = get_database_manager()

        # Test that errors in session don't break global manager
        with pytest.raises(ValueError):
            async with manager.get_session() as session:
                player_repo = TrackedPlayerRepository(session)
                await player_repo.create("ErrorTest", "NA1")
                raise ValueError("Test error")

        # Manager should still be usable after error
        async with manager.get_session() as session:
            result = await session.execute(text("SELECT 1"))
            assert result.scalar() == 1

        await close_database()


@pytest.mark.integration
class TestDatabaseConfiguration:
    """Test suite for database configuration handling."""

    @pytest.mark.asyncio
    async def test_config_with_debug_logging(self):
        """Test database manager with debug logging enabled."""
        config = Config(
            database_url="postgresql+asyncpg://test:test@localhost/test",
            database_name="test",
            riot_api_key="test",
            log_level="DEBUG",
            environment=Environment.CI,
        )

        manager = DatabaseManager(config)

        # Should initialize without error
        # Note: We can't test actual logging output easily, but this verifies
        # the configuration is handled correctly
        assert manager.config.log_level == "DEBUG"

    @pytest.mark.asyncio
    async def test_config_with_production_settings(self):
        """Test database manager with production-like configuration."""
        config = Config(
            database_url="postgresql+asyncpg://test:test@localhost/test",
            database_name="test",
            riot_api_key="test",
            environment=Environment.PRODUCTION,
            log_level="INFO",
        )

        manager = DatabaseManager(config)
        assert manager.config.environment == Environment.PRODUCTION
        assert manager.config.log_level == "INFO"
