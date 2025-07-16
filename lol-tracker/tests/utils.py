"""Test utility functions."""
from typing import List
from sqlalchemy import select
from sqlalchemy.ext.asyncio import AsyncSession

from lol_tracker.database.models import TrackedPlayer, GameState


async def count_tracked_players(session: AsyncSession) -> int:
    """Count the number of tracked players in the database."""
    result = await session.execute(select(TrackedPlayer))
    return len(result.scalars().all())


async def count_game_states(session: AsyncSession) -> int:
    """Count the number of game states in the database."""
    result = await session.execute(select(GameState))
    return len(result.scalars().all())


async def get_all_tracked_players(session: AsyncSession) -> List[TrackedPlayer]:
    """Get all tracked players from the database."""
    result = await session.execute(select(TrackedPlayer))
    return list(result.scalars().all())


async def get_all_game_states(session: AsyncSession) -> List[GameState]:
    """Get all game states from the database."""
    result = await session.execute(select(GameState))
    return list(result.scalars().all())


async def get_game_states_for_player(session: AsyncSession, player_id: int) -> List[GameState]:
    """Get all game states for a specific player."""
    result = await session.execute(
        select(GameState).where(GameState.player_id == player_id)
    )
    return list(result.scalars().all())


def assert_player_equals(actual: TrackedPlayer, expected: TrackedPlayer, ignore_id: bool = True):
    """Assert that two TrackedPlayer objects are equal."""
    if not ignore_id:
        assert actual.id == expected.id
    
    assert actual.summoner_name == expected.summoner_name
    assert actual.region == expected.region
    assert actual.puuid == expected.puuid
    assert actual.account_id == expected.account_id
    assert actual.summoner_id == expected.summoner_id
    assert actual.is_active == expected.is_active


def assert_game_state_equals(actual: GameState, expected: GameState, ignore_id: bool = True):
    """Assert that two GameState objects are equal."""
    if not ignore_id:
        assert actual.id == expected.id
    
    assert actual.player_id == expected.player_id
    assert actual.status == expected.status
    assert actual.game_id == expected.game_id
    assert actual.queue_type == expected.queue_type
    assert actual.won == expected.won
    assert actual.duration_seconds == expected.duration_seconds
    assert actual.champion_played == expected.champion_played