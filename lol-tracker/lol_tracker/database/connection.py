"""Database connection and session management for LoL Tracker service."""

import logging
from contextlib import asynccontextmanager
from typing import AsyncGenerator, Optional

from sqlalchemy.ext.asyncio import (
    AsyncSession,
    AsyncEngine,
    create_async_engine,
    async_sessionmaker,
)
from sqlalchemy.pool import NullPool

from lol_tracker.config import Config
from lol_tracker.database.models import Base

logger = logging.getLogger(__name__)


class DatabaseManager:
    """Manages database connection and session lifecycle."""

    def __init__(self, config: Config):
        self.config = config
        self._engine: Optional[AsyncEngine] = None
        self._session_factory: Optional[async_sessionmaker[AsyncSession]] = None

    async def initialize(self) -> None:
        """Initialize database engine and session factory."""
        if self._engine is not None:
            logger.warning("Database manager already initialized")
            return

        # Create async engine with proper connection pooling
        self._engine = create_async_engine(
            self.config.get_database_url(),
            echo=self.config.log_level == "DEBUG",
            poolclass=NullPool,  # Use NullPool for better connection management in async context
            pool_pre_ping=True,  # Verify connections before use
        )

        # Create session factory
        self._session_factory = async_sessionmaker(
            bind=self._engine,
            class_=AsyncSession,
            expire_on_commit=False,
        )

        logger.info("Database manager initialized successfully")

    async def close(self) -> None:
        """Close database engine and clean up resources."""
        if self._engine is not None:
            await self._engine.dispose()
            self._engine = None
            self._session_factory = None
            logger.info("Database manager closed")

    @asynccontextmanager
    async def get_session(self) -> AsyncGenerator[AsyncSession, None]:
        """Get a database session with automatic cleanup."""
        if self._session_factory is None:
            raise RuntimeError(
                "Database manager not initialized. Call initialize() first."
            )

        async with self._session_factory() as session:
            try:
                yield session
            except Exception:
                await session.rollback()
                raise
            finally:
                await session.close()

    async def create_tables(self) -> None:
        """Create all database tables. Used for testing and initial setup."""
        if self._engine is None:
            raise RuntimeError(
                "Database manager not initialized. Call initialize() first."
            )

        async with self._engine.begin() as conn:
            await conn.run_sync(Base.metadata.create_all)

        logger.info("Database tables created successfully")

    async def drop_tables(self) -> None:
        """Drop all database tables. Used for testing cleanup."""
        if self._engine is None:
            raise RuntimeError(
                "Database manager not initialized. Call initialize() first."
            )

        async with self._engine.begin() as conn:
            await conn.run_sync(Base.metadata.drop_all)

        logger.info("Database tables dropped successfully")

    @property
    def engine(self) -> AsyncEngine:
        """Get the database engine."""
        if self._engine is None:
            raise RuntimeError(
                "Database manager not initialized. Call initialize() first."
            )
        return self._engine


# Global database manager instance
_db_manager: Optional[DatabaseManager] = None


def get_database_manager() -> DatabaseManager:
    """Get the global database manager instance."""
    global _db_manager
    if _db_manager is None:
        raise RuntimeError(
            "Database manager not initialized. Call initialize_database() first."
        )
    return _db_manager


async def initialize_database(config: Config) -> DatabaseManager:
    """Initialize the global database manager."""
    global _db_manager
    if _db_manager is not None:
        logger.warning("Database manager already initialized")
        return _db_manager

    _db_manager = DatabaseManager(config)
    await _db_manager.initialize()
    return _db_manager


async def close_database() -> None:
    """Close the global database manager."""
    global _db_manager
    if _db_manager is not None:
        await _db_manager.close()
        _db_manager = None
