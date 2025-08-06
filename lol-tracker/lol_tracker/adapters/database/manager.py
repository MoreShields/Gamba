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
from .models import Base, TrackedPlayer as TrackedPlayerModel, GameState as GameStateModel
from ...core.entities import Player, GameState, LoLGameResult
from ...core.enums import GameStatus, QueueType

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

    # Conversion methods
    def _convert_db_player_to_core_entity(self, player_record: TrackedPlayerModel) -> Player:
        """Convert database TrackedPlayer model to core Player entity.
        
        Args:
            player_record: SQLAlchemy TrackedPlayer instance
            
        Returns:
            Core Player entity
        """
        return Player(
            game_name=player_record.game_name,
            tag_line=player_record.tag_line,
            puuid=player_record.puuid,
            id=player_record.id,
            created_at=player_record.created_at,
            updated_at=player_record.updated_at
        )
    
    def _convert_db_gamestate_to_core_entity(self, gamestate_record: GameStateModel) -> GameState:
        """Convert database GameState model to core GameState entity.
        
        Args:
            gamestate_record: SQLAlchemy GameState instance
            
        Returns:
            Core GameState entity
        """
        # Create game result if we have result data
        game_result = None
        if (gamestate_record.won is not None and 
            gamestate_record.duration_seconds is not None and 
            gamestate_record.champion_played is not None):
            game_result = LoLGameResult(
                won=gamestate_record.won,
                duration_seconds=gamestate_record.duration_seconds,
                champion_played=gamestate_record.champion_played
            )
        
        return GameState(
            status=GameStatus(gamestate_record.status),
            player_id=gamestate_record.player_id,
            game_id=gamestate_record.game_id,
            queue_type=QueueType(gamestate_record.queue_type) if gamestate_record.queue_type else None,
            game_result=game_result,
            created_at=gamestate_record.created_at,
            game_start_time=gamestate_record.game_start_time,
            game_end_time=gamestate_record.game_end_time,
            id=gamestate_record.id
        )

    # TrackedPlayer repository methods
    async def create_tracked_player(
        self,
        game_name: str,
        tag_line: str,
        puuid: str,
    ) -> Player:
        """Create a new tracked player."""
        async with self.get_session() as session:
            player = TrackedPlayerModel(
                game_name=game_name,
                tag_line=tag_line,
                puuid=puuid,
            )
            session.add(player)
            await session.commit()
            await session.refresh(player)
            return self._convert_db_player_to_core_entity(player)

    async def get_tracked_player_by_puuid(self, puuid: str) -> Optional[Player]:
        """Get a tracked player by PUUID."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedPlayerModel).where(TrackedPlayerModel.puuid == puuid)
            )
            player_record = result.scalar_one_or_none()
            return self._convert_db_player_to_core_entity(player_record) if player_record else None

    async def get_all_players(self) -> List[Player]:
        """Get all tracked players."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedPlayerModel)
            )
            player_records = result.scalars().all()
            return [self._convert_db_player_to_core_entity(p) for p in player_records]

    async def delete_tracked_player(self, player_id: int) -> bool:
        """Delete a tracked player."""
        async with self.get_session() as session:
            result = await session.execute(
                delete(TrackedPlayerModel).where(TrackedPlayerModel.id == player_id)
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
            game_state = GameStateModel(
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
            return self._convert_db_gamestate_to_core_entity(game_state)

    async def get_latest_game_state_for_player(self, player_id: int) -> Optional[GameState]:
        """Get the latest game state for a player."""
        async with self.get_session() as session:
            result = await session.execute(
                select(GameStateModel)
                .where(GameStateModel.player_id == player_id)
                .order_by(GameStateModel.created_at.desc())
                .limit(1)
            )
            game_state_record = result.scalar_one_or_none()
            return self._convert_db_gamestate_to_core_entity(game_state_record) if game_state_record else None

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
                update(GameStateModel)
                .where(GameStateModel.id == game_state_id)
                .values(
                    won=won,
                    duration_seconds=duration_seconds,
                    champion_played=champion_played,
                    game_end_time=game_end_time or datetime.utcnow(),
                )
            )
            await session.commit()
            return result.rowcount > 0
