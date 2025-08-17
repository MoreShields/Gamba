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
from sqlalchemy import select, update, delete, func
from sqlalchemy.orm import selectinload

from ...config import Config
from .models import Base, TrackedPlayer as TrackedPlayerModel, GameState as GameStateModel, TrackedGame as TrackedGameModel
from ...core.entities import Player, GameState
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
            # Note: puuid field removed from Player entity
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
        # Parse queue type
        queue_type = None
        if gamestate_record.queue_type:
            queue_type = QueueType.from_string(gamestate_record.queue_type)
        
        # Let the domain entity handle game result parsing
        game_result = GameState.parse_game_result(
            queue_type, 
            gamestate_record.game_result_data
        )
        
        return GameState(
            status=GameStatus(gamestate_record.status),
            player_id=gamestate_record.player_id,
            game_id=gamestate_record.game_id,
            queue_type=queue_type,
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
        tag_line: str
    ) -> Player:
        """Create a new tracked player.
        
        Args:
            game_name: Player's game name
            tag_line: Player's tag line
        """
        async with self.get_session() as session:
            player = TrackedPlayerModel(
                game_name=game_name,
                tag_line=tag_line
            )
            session.add(player)
            await session.commit()
            await session.refresh(player)
            return self._convert_db_player_to_core_entity(player)

    # Note: get_tracked_player_by_puuid removed - use get_tracked_player_by_riot_id instead
    
    async def get_tracked_player_by_riot_id(self, game_name: str, tag_line: str) -> Optional[Player]:
        """Get a tracked player by Riot ID (game name and tag line)."""
        async with self.get_session() as session:
            # Case-insensitive comparison for game_name and tag_line
            result = await session.execute(
                select(TrackedPlayerModel).where(
                    func.lower(TrackedPlayerModel.game_name) == game_name.lower(),
                    func.lower(TrackedPlayerModel.tag_line) == tag_line.lower()
                )
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
        game_result_data: dict,
        game_end_time: Optional[datetime] = None,
    ) -> bool:
        """Update game result information.
        
        Args:
            game_state_id: ID of the game state to update
            game_result_data: Dictionary containing game result data
                             (will be stored as JSON)
            game_end_time: Optional game end time
        
        Returns:
            True if update was successful
        """
        async with self.get_session() as session:
            update_values = {
                "game_result_data": game_result_data,
                "duration_seconds": game_result_data.get("duration_seconds"),
                "game_end_time": game_end_time or datetime.utcnow(),
            }
            
            result = await session.execute(
                update(GameStateModel)
                .where(GameStateModel.id == game_state_id)
                .values(**update_values)
            )
            await session.commit()
            return result.rowcount > 0
    
    # TrackedGame repository methods (game-centric model)
    
    async def get_tracked_game(
        self, 
        player_id: int, 
        game_id: str
    ) -> Optional[TrackedGameModel]:
        """Get a tracked game by player and game ID."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedGameModel)
                .where(
                    TrackedGameModel.player_id == player_id,
                    TrackedGameModel.game_id == game_id
                )
            )
            return result.scalar_one_or_none()
    
    async def create_tracked_game(
        self,
        player_id: int,
        game_id: str,
        game_type: str,
        status: str = 'ACTIVE',
        queue_type: Optional[str] = None,
        started_at: Optional[datetime] = None,
        raw_api_response: Optional[str] = None
    ) -> TrackedGameModel:
        """Create a new tracked game entry."""
        async with self.get_session() as session:
            game = TrackedGameModel(
                player_id=player_id,
                game_id=game_id,
                game_type=game_type,
                status=status,
                queue_type=queue_type,
                started_at=started_at,
                raw_api_response=raw_api_response
            )
            session.add(game)
            await session.commit()
            await session.refresh(game)
            return game
    
    async def get_games_by_status(self, status: str) -> List[TrackedGameModel]:
        """Get all games with a specific status."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedGameModel)
                .where(TrackedGameModel.status == status)
                .order_by(TrackedGameModel.detected_at)
            )
            return list(result.scalars().all())
    
    async def complete_tracked_game(
        self,
        game_id: int,
        game_result_data: Optional[dict] = None,
        duration_seconds: Optional[int] = None,
        completed_at: Optional[datetime] = None
    ) -> bool:
        """Mark a game as completed with results."""
        async with self.get_session() as session:
            result = await session.execute(
                update(TrackedGameModel)
                .where(TrackedGameModel.id == game_id)
                .values(
                    status='COMPLETED',
                    game_result_data=game_result_data,
                    duration_seconds=duration_seconds,
                    completed_at=completed_at or datetime.utcnow(),
                    last_error=None  # Clear any previous errors
                )
            )
            await session.commit()
            return result.rowcount > 0
    
    async def update_game_error(
        self, 
        player_id: int,
        game_id: str, 
        error: str
    ) -> bool:
        """Update the last error for a game."""
        async with self.get_session() as session:
            result = await session.execute(
                update(TrackedGameModel)
                .where(
                    TrackedGameModel.player_id == player_id,
                    TrackedGameModel.game_id == game_id
                )
                .values(last_error=error)
            )
            await session.commit()
            return result.rowcount > 0
    
    async def get_player_by_id(self, player_id: int) -> Optional[Player]:
        """Get a player by their database ID."""
        async with self.get_session() as session:
            result = await session.execute(
                select(TrackedPlayerModel)
                .where(TrackedPlayerModel.id == player_id)
            )
            player_record = result.scalar_one_or_none()
            return self._convert_db_player_to_core_entity(player_record) if player_record else None
