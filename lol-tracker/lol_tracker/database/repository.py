"""Repository classes for database operations."""

from typing import List, Optional
from datetime import datetime

from sqlalchemy import select, update, delete
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.orm import selectinload

from lol_tracker.database.models import TrackedPlayer, GameState


class TrackedPlayerRepository:
    """Repository for TrackedPlayer operations."""

    def __init__(self, session: AsyncSession):
        self.session = session

    async def create(
        self,
        game_name: str,
        tag_line: str,
        puuid: str,
    ) -> TrackedPlayer:
        """Create a new tracked player."""
        player = TrackedPlayer(
            game_name=game_name,
            tag_line=tag_line,
            puuid=puuid,
        )
        self.session.add(player)
        await self.session.flush()
        return player

    async def get_by_id(self, player_id: int) -> Optional[TrackedPlayer]:
        """Get a tracked player by ID."""
        result = await self.session.execute(
            select(TrackedPlayer).where(TrackedPlayer.id == player_id)
        )
        return result.scalar_one_or_none()


    async def get_by_puuid(self, puuid: str) -> Optional[TrackedPlayer]:
        """Get a tracked player by PUUID."""
        result = await self.session.execute(
            select(TrackedPlayer).where(TrackedPlayer.puuid == puuid)
        )
        return result.scalar_one_or_none()

    async def get_all_active(self) -> List[TrackedPlayer]:
        """Get all active tracked players."""
        result = await self.session.execute(
            select(TrackedPlayer).where(TrackedPlayer.is_active == True)
        )
        return list(result.scalars().all())

    async def update_puuid(
        self,
        player_id: int,
        puuid: str,
    ) -> bool:
        """Update PUUID for a tracked player."""
        result = await self.session.execute(
            update(TrackedPlayer)
            .where(TrackedPlayer.id == player_id)
            .values(puuid=puuid, updated_at=datetime.utcnow())
        )
        return result.rowcount > 0

    async def set_active_status(self, player_id: int, is_active: bool) -> bool:
        """Set the active status of a tracked player."""
        result = await self.session.execute(
            update(TrackedPlayer)
            .where(TrackedPlayer.id == player_id)
            .values(is_active=is_active, updated_at=datetime.utcnow())
        )
        return result.rowcount > 0

    async def delete(self, player_id: int) -> bool:
        """Delete a tracked player."""
        result = await self.session.execute(
            delete(TrackedPlayer).where(TrackedPlayer.id == player_id)
        )
        return result.rowcount > 0


class GameStateRepository:
    """Repository for GameState operations."""

    def __init__(self, session: AsyncSession):
        self.session = session

    async def create(
        self,
        player_id: int,
        status: str,
        game_id: Optional[str] = None,
        queue_type: Optional[str] = None,
        game_start_time: Optional[datetime] = None,
        raw_api_response: Optional[str] = None,
    ) -> GameState:
        """Create a new game state record."""
        game_state = GameState(
            player_id=player_id,
            status=status,
            game_id=game_id,
            queue_type=queue_type,
            game_start_time=game_start_time,
            raw_api_response=raw_api_response,
        )
        self.session.add(game_state)
        await self.session.flush()
        return game_state

    async def get_latest_for_player(self, player_id: int) -> Optional[GameState]:
        """Get the latest game state for a player."""
        result = await self.session.execute(
            select(GameState)
            .where(GameState.player_id == player_id)
            .order_by(GameState.created_at.desc())
            .limit(1)
        )
        return result.scalar_one_or_none()

    async def get_by_game_id(self, game_id: str) -> List[GameState]:
        """Get all game states for a specific game ID."""
        result = await self.session.execute(
            select(GameState).where(GameState.game_id == game_id)
        )
        return list(result.scalars().all())

    async def update_game_result(
        self,
        game_state_id: int,
        won: bool,
        duration_seconds: int,
        champion_played: str,
        game_end_time: Optional[datetime] = None,
    ) -> bool:
        """Update game result information."""
        result = await self.session.execute(
            update(GameState)
            .where(GameState.id == game_state_id)
            .values(
                won=won,
                duration_seconds=duration_seconds,
                champion_played=champion_played,
                game_end_time=game_end_time or datetime.utcnow(),
            )
        )
        return result.rowcount > 0

    async def get_recent_games_for_player(
        self, player_id: int, limit: int = 10
    ) -> List[GameState]:
        """Get recent game states for a player."""
        result = await self.session.execute(
            select(GameState)
            .where(GameState.player_id == player_id)
            .order_by(GameState.created_at.desc())
            .limit(limit)
        )
        return list(result.scalars().all())

    async def get_active_games(self) -> List[GameState]:
        """Get all currently active game states (IN_GAME status)."""
        result = await self.session.execute(
            select(GameState)
            .options(selectinload(GameState.player))
            .where(GameState.status == "IN_GAME")
            .order_by(GameState.created_at.desc())
        )
        return list(result.scalars().all())

    async def delete_old_states(self, player_id: int, keep_count: int = 100) -> int:
        """Delete old game state records, keeping the most recent ones."""
        # Get the IDs of records to keep
        keep_result = await self.session.execute(
            select(GameState.id)
            .where(GameState.player_id == player_id)
            .order_by(GameState.created_at.desc())
            .limit(keep_count)
        )
        keep_ids = [row[0] for row in keep_result.fetchall()]

        if not keep_ids:
            return 0

        # Delete records not in the keep list
        result = await self.session.execute(
            delete(GameState).where(
                GameState.player_id == player_id, GameState.id.notin_(keep_ids)
            )
        )
        return result.rowcount
