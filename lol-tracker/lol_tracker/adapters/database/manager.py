"""Simplified database infrastructure layer."""

import logging
from contextlib import asynccontextmanager
from typing import AsyncGenerator, Optional, List
from datetime import datetime

from sqlalchemy.ext.asyncio import (
    AsyncSession,
    AsyncEngine,
    create_async_engine,
    async_sessionmaker,
)
from sqlalchemy.pool import NullPool
from sqlalchemy import select, update, delete
from sqlalchemy.orm import selectinload

from ...config import Config
from .models import Base, TrackedPlayer, GameState

logger = logging.getLogger(__name__)


class DatabaseManager:
    """Manages database connection and provides direct repository methods."""

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

    def get_session_factory(self) -> async_sessionmaker[AsyncSession]:
        """Get the session factory for use with other components.
        
        Returns:
            The SQLAlchemy async session factory
            
        Raises:
            RuntimeError: If database manager is not initialized
        """
        if self._session_factory is None:
            raise RuntimeError(
                "Database manager not initialized. Call initialize() first."
            )
        
        return self._session_factory

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

    # TrackedPlayer repository methods
    async def create_tracked_player(
        self,
        game_name: str,
        tag_line: str,
        puuid: str,
    ) -> TrackedPlayer:
        """Create a new tracked player."""
        async with self.get_session() as session:
            player = TrackedPlayer(
                game_name=game_name,
                tag_line=tag_line,
                puuid=puuid,
            )
            session.add(player)
            await session.commit()
            await session.refresh(player)
            return player

    async def get_tracked_player_by_puuid(self, puuid: str) -> Optional[TrackedPlayer]:
        """Get a tracked player by PUUID."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedPlayer).where(TrackedPlayer.puuid == puuid)
            )
            return result.scalar_one_or_none()

    async def get_all_players(self) -> List[TrackedPlayer]:
        """Get all tracked players."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedPlayer)
            )
            return list(result.scalars().all())

    async def delete_tracked_player(self, player_id: int) -> bool:
        """Delete a tracked player."""
        async with self.get_session() as session:
            result = await session.execute(
                delete(TrackedPlayer).where(TrackedPlayer.id == player_id)
            )
            await session.commit()
            return result.rowcount > 0

    # GameState repository methods
    async def create_game_state(
        self,
        player_id: int,
        status: str,
        game_id: Optional[str] = None,
        queue_type: Optional[str] = None,
        game_start_time: Optional[datetime] = None,
        raw_api_response: Optional[str] = None,
    ) -> GameState:
        """Create a new game state record."""
        async with self.get_session() as session:
            game_state = GameState(
                player_id=player_id,
                status=status,
                game_id=game_id,
                queue_type=queue_type,
                game_start_time=game_start_time,
                raw_api_response=raw_api_response,
            )
            session.add(game_state)
            await session.commit()
            await session.refresh(game_state)
            return game_state

    async def get_latest_game_state_for_player(self, player_id: int) -> Optional[GameState]:
        """Get the latest game state for a player."""
        async with self.get_session() as session:
            result = await session.execute(
                select(GameState)
                .where(GameState.player_id == player_id)
                .order_by(GameState.created_at.desc())
                .limit(1)
            )
            return result.scalar_one_or_none()

    async def update_game_result(
        self,
        game_state_id: int,
        won: bool,
        duration_seconds: int,
        champion_played: str,
        game_end_time: Optional[datetime] = None,
    ) -> bool:
        """Update game result information."""
        async with self.get_session() as session:
            result = await session.execute(
                update(GameState)
                .where(GameState.id == game_state_id)
                .values(
                    won=won,
                    duration_seconds=duration_seconds,
                    champion_played=champion_played,
                    game_end_time=game_end_time or datetime.utcnow(),
                )
            )
            await session.commit()
            return result.rowcount > 0

    async def get_recent_games_for_player(
        self, player_id: int, limit: int = 10
    ) -> List[GameState]:
        """Get recent game states for a player."""
        async with self.get_session() as session:
            result = await session.execute(
                select(GameState)
                .where(GameState.player_id == player_id)
                .order_by(GameState.created_at.desc())
                .limit(limit)
            )
            return list(result.scalars().all())

    async def get_active_games(self) -> List[GameState]:
        """Get all currently active game states (IN_GAME status)."""
        async with self.get_session() as session:
            result = await session.execute(
                select(GameState)
                .options(selectinload(GameState.player))
                .where(GameState.status == "IN_GAME")
                .order_by(GameState.created_at.desc())
            )
            return list(result.scalars().all())


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