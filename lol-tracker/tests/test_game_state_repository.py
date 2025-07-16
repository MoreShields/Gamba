"""Integration tests for GameStateRepository."""
import pytest
import pytest_asyncio
from datetime import datetime, timedelta
from sqlalchemy.ext.asyncio import AsyncSession

from lol_tracker.database.repository import TrackedPlayerRepository, GameStateRepository
from lol_tracker.database.models import GameState
from tests.factories import TrackedPlayerFactory, GameStateFactory
from tests.utils import count_game_states, assert_game_state_equals, get_game_states_for_player


@pytest.mark.integration
class TestGameStateRepository:
    """Test suite for GameStateRepository integration tests."""

    @pytest_asyncio.fixture
    async def tracked_player(self, db_session: AsyncSession):
        """Create a tracked player for testing game states."""
        player_repo = TrackedPlayerRepository(db_session)
        return await player_repo.create("TestPlayer", "NA1", puuid="test_puuid")

    @pytest.mark.asyncio
    async def test_create_game_state(self, db_session: AsyncSession, tracked_player):
        """Test creating a new game state."""
        repo = GameStateRepository(db_session)
        
        game_state = await repo.create(
            player_id=tracked_player.id,
            status="IN_CHAMPION_SELECT",
            game_id="test_game_123",
            queue_type="RANKED_SOLO_5x5",
            game_start_time=datetime.utcnow(),
            raw_api_response='{"test": "data"}',
        )
        
        assert game_state.id is not None
        assert game_state.player_id == tracked_player.id
        assert game_state.status == "IN_CHAMPION_SELECT"
        assert game_state.game_id == "test_game_123"
        assert game_state.queue_type == "RANKED_SOLO_5x5"
        assert game_state.game_start_time is not None
        assert game_state.raw_api_response == '{"test": "data"}'
        assert isinstance(game_state.created_at, datetime)

    @pytest.mark.asyncio
    async def test_create_game_state_minimal(self, db_session: AsyncSession, tracked_player):
        """Test creating a game state with minimal required data."""
        repo = GameStateRepository(db_session)
        
        game_state = await repo.create(
            player_id=tracked_player.id,
            status="NOT_IN_GAME",
        )
        
        assert game_state.id is not None
        assert game_state.player_id == tracked_player.id
        assert game_state.status == "NOT_IN_GAME"
        assert game_state.game_id is None
        assert game_state.queue_type is None
        assert game_state.game_start_time is None

    @pytest.mark.asyncio
    async def test_get_latest_for_player(self, db_session: AsyncSession, tracked_player):
        """Test retrieving the latest game state for a player."""
        repo = GameStateRepository(db_session)
        
        # Create multiple game states for the same player
        state1 = await repo.create(tracked_player.id, "NOT_IN_GAME")
        await asyncio.sleep(0.01)  # Ensure different timestamps
        state2 = await repo.create(tracked_player.id, "IN_CHAMPION_SELECT")
        await asyncio.sleep(0.01)
        state3 = await repo.create(tracked_player.id, "IN_GAME")
        
        # Get the latest state
        latest_state = await repo.get_latest_for_player(tracked_player.id)
        
        assert latest_state is not None
        assert latest_state.id == state3.id
        assert latest_state.status == "IN_GAME"

    @pytest.mark.asyncio
    async def test_get_latest_for_player_not_found(self, db_session: AsyncSession):
        """Test retrieving latest game state for a non-existent player."""
        repo = GameStateRepository(db_session)
        
        latest_state = await repo.get_latest_for_player(99999)
        
        assert latest_state is None

    @pytest.mark.asyncio
    async def test_get_by_game_id(self, db_session: AsyncSession, tracked_player):
        """Test retrieving game states by game ID."""
        repo = GameStateRepository(db_session)
        player_repo = TrackedPlayerRepository(db_session)
        
        # Create another player
        player2 = await player_repo.create("TestPlayer2", "EUW1")
        
        # Create game states for the same game ID
        state1 = await repo.create(
            tracked_player.id, 
            "IN_GAME", 
            game_id="shared_game_123"
        )
        state2 = await repo.create(
            player2.id, 
            "IN_GAME", 
            game_id="shared_game_123"
        )
        # Create a state with different game ID
        await repo.create(tracked_player.id, "NOT_IN_GAME", game_id="different_game")
        
        # Get states by game ID
        game_states = await repo.get_by_game_id("shared_game_123")
        
        assert len(game_states) == 2
        game_state_ids = {state.id for state in game_states}
        assert game_state_ids == {state1.id, state2.id}

    @pytest.mark.asyncio
    async def test_get_by_game_id_not_found(self, db_session: AsyncSession):
        """Test retrieving game states for a non-existent game ID."""
        repo = GameStateRepository(db_session)
        
        game_states = await repo.get_by_game_id("non_existent_game")
        
        assert game_states == []

    @pytest.mark.asyncio
    async def test_update_game_result(self, db_session: AsyncSession, tracked_player):
        """Test updating game result information."""
        repo = GameStateRepository(db_session)
        
        # Create a game state
        game_state = await repo.create(
            tracked_player.id,
            "IN_GAME",
            game_id="result_test_game"
        )
        
        end_time = datetime.utcnow()
        
        # Update the game result
        success = await repo.update_game_result(
            game_state.id,
            won=True,
            duration_seconds=1800,
            champion_played="Jinx",
            game_end_time=end_time,
        )
        
        assert success is True
        
        # Verify the update
        updated_states = await repo.get_by_game_id("result_test_game")
        updated_state = updated_states[0]
        
        assert updated_state.won is True
        assert updated_state.duration_seconds == 1800
        assert updated_state.champion_played == "Jinx"
        assert updated_state.game_end_time == end_time

    @pytest.mark.asyncio
    async def test_update_game_result_auto_end_time(self, db_session: AsyncSession, tracked_player):
        """Test updating game result with automatic end time."""
        repo = GameStateRepository(db_session)
        
        # Create a game state
        game_state = await repo.create(tracked_player.id, "IN_GAME")
        
        before_update = datetime.utcnow()
        
        # Update without providing end time
        success = await repo.update_game_result(
            game_state.id,
            won=False,
            duration_seconds=900,
            champion_played="Yasuo",
        )
        
        assert success is True
        
        # Verify automatic end time was set
        updated_state = await repo.get_latest_for_player(tracked_player.id)
        assert updated_state.game_end_time >= before_update
        assert updated_state.won is False
        assert updated_state.duration_seconds == 900
        assert updated_state.champion_played == "Yasuo"

    @pytest.mark.asyncio
    async def test_update_game_result_nonexistent(self, db_session: AsyncSession):
        """Test updating game result for a non-existent game state."""
        repo = GameStateRepository(db_session)
        
        success = await repo.update_game_result(
            99999,
            won=True,
            duration_seconds=1000,
            champion_played="TestChampion",
        )
        
        assert success is False

    @pytest.mark.asyncio
    async def test_get_recent_games_for_player(self, db_session: AsyncSession, tracked_player):
        """Test retrieving recent games for a player."""
        repo = GameStateRepository(db_session)
        
        # Create multiple game states
        states = []
        for i in range(15):
            state = await repo.create(
                tracked_player.id,
                "NOT_IN_GAME",
                game_id=f"game_{i}"
            )
            states.append(state)
            if i < 14:  # Don't sleep after the last one
                await asyncio.sleep(0.01)
        
        # Get recent games with default limit (10)
        recent_games = await repo.get_recent_games_for_player(tracked_player.id)
        
        assert len(recent_games) == 10
        
        # Verify they are the most recent ones (last 10 created)
        expected_ids = {state.id for state in states[-10:]}
        actual_ids = {state.id for state in recent_games}
        assert actual_ids == expected_ids

    @pytest.mark.asyncio
    async def test_get_recent_games_with_custom_limit(self, db_session: AsyncSession, tracked_player):
        """Test retrieving recent games with custom limit."""
        repo = GameStateRepository(db_session)
        
        # Create 5 game states
        for i in range(5):
            await repo.create(tracked_player.id, "NOT_IN_GAME", game_id=f"game_{i}")
            if i < 4:
                await asyncio.sleep(0.01)
        
        # Get recent games with limit of 3
        recent_games = await repo.get_recent_games_for_player(tracked_player.id, limit=3)
        
        assert len(recent_games) == 3

    @pytest.mark.asyncio
    async def test_get_recent_games_empty(self, db_session: AsyncSession, tracked_player):
        """Test retrieving recent games when none exist."""
        repo = GameStateRepository(db_session)
        
        recent_games = await repo.get_recent_games_for_player(tracked_player.id)
        
        assert recent_games == []

    @pytest.mark.asyncio
    async def test_get_active_games(self, db_session: AsyncSession, tracked_player):
        """Test retrieving all active game states."""
        repo = GameStateRepository(db_session)
        player_repo = TrackedPlayerRepository(db_session)
        
        # Create another player
        player2 = await player_repo.create("ActiveTest2", "EUW1")
        
        # Create various game states
        await repo.create(tracked_player.id, "NOT_IN_GAME")
        active_state1 = await repo.create(tracked_player.id, "IN_GAME", game_id="active_1")
        await repo.create(tracked_player.id, "IN_CHAMPION_SELECT")
        active_state2 = await repo.create(player2.id, "IN_GAME", game_id="active_2")
        
        # Get active games
        active_games = await repo.get_active_games()
        
        assert len(active_games) == 2
        active_ids = {state.id for state in active_games}
        assert active_ids == {active_state1.id, active_state2.id}
        
        # Verify player data is loaded
        for state in active_games:
            assert state.player is not None
            assert state.player.summoner_name in ["TestPlayer", "ActiveTest2"]

    @pytest.mark.asyncio
    async def test_get_active_games_empty(self, db_session: AsyncSession):
        """Test retrieving active games when none exist."""
        repo = GameStateRepository(db_session)
        
        active_games = await repo.get_active_games()
        
        assert active_games == []

    @pytest.mark.asyncio
    async def test_delete_old_states(self, db_session: AsyncSession, tracked_player):
        """Test deleting old game state records."""
        repo = GameStateRepository(db_session)
        
        # Create 15 game states
        created_states = []
        for i in range(15):
            state = await repo.create(
                tracked_player.id,
                "NOT_IN_GAME",
                game_id=f"old_game_{i}"
            )
            created_states.append(state)
            if i < 14:
                await asyncio.sleep(0.01)
        
        # Delete old states, keeping only 5 most recent
        deleted_count = await repo.delete_old_states(tracked_player.id, keep_count=5)
        
        assert deleted_count == 10
        
        # Verify only 5 states remain
        remaining_states = await get_game_states_for_player(db_session, tracked_player.id)
        assert len(remaining_states) == 5
        
        # Verify the remaining states are the most recent ones
        remaining_ids = {state.id for state in remaining_states}
        expected_ids = {state.id for state in created_states[-5:]}
        assert remaining_ids == expected_ids

    @pytest.mark.asyncio
    async def test_delete_old_states_keep_all(self, db_session: AsyncSession, tracked_player):
        """Test deleting old states when keep_count is larger than existing states."""
        repo = GameStateRepository(db_session)
        
        # Create 3 game states
        for i in range(3):
            await repo.create(tracked_player.id, "NOT_IN_GAME", game_id=f"keep_game_{i}")
        
        # Try to delete old states, keeping 10 (more than exist)
        deleted_count = await repo.delete_old_states(tracked_player.id, keep_count=10)
        
        assert deleted_count == 0
        
        # Verify all states remain
        remaining_states = await get_game_states_for_player(db_session, tracked_player.id)
        assert len(remaining_states) == 3

    @pytest.mark.asyncio
    async def test_delete_old_states_no_states(self, db_session: AsyncSession, tracked_player):
        """Test deleting old states when no states exist."""
        repo = GameStateRepository(db_session)
        
        deleted_count = await repo.delete_old_states(tracked_player.id, keep_count=5)
        
        assert deleted_count == 0

    @pytest.mark.asyncio
    async def test_game_state_sequence(self, db_session: AsyncSession, tracked_player):
        """Test a typical game state sequence (champion select -> in game -> finished)."""
        repo = GameStateRepository(db_session)
        
        # Start with champion select
        champion_select = await repo.create(
            tracked_player.id,
            "IN_CHAMPION_SELECT",
            game_id="sequence_game",
            queue_type="RANKED_SOLO_5x5"
        )
        
        # Move to in game
        in_game = await repo.create(
            tracked_player.id,
            "IN_GAME",
            game_id="sequence_game",
            queue_type="RANKED_SOLO_5x5",
            game_start_time=datetime.utcnow()
        )
        
        # Finish the game
        finished = await repo.create(
            tracked_player.id,
            "NOT_IN_GAME"
        )
        
        # Update the in-game state with results
        await repo.update_game_result(
            in_game.id,
            won=True,
            duration_seconds=2100,
            champion_played="Zed"
        )
        
        # Verify the sequence
        latest_state = await repo.get_latest_for_player(tracked_player.id)
        assert latest_state.id == finished.id
        assert latest_state.status == "NOT_IN_GAME"
        
        # Verify the game result was recorded
        game_states = await repo.get_by_game_id("sequence_game")
        in_game_state = next(state for state in game_states if state.status == "IN_GAME")
        assert in_game_state.won is True
        assert in_game_state.duration_seconds == 2100
        assert in_game_state.champion_played == "Zed"

    @pytest.mark.asyncio
    async def test_multiple_players_same_game(self, db_session: AsyncSession):
        """Test tracking multiple players in the same game."""
        repo = GameStateRepository(db_session)
        player_repo = TrackedPlayerRepository(db_session)
        
        # Create multiple players
        player1 = await player_repo.create("Player1", "NA1")
        player2 = await player_repo.create("Player2", "NA1")
        player3 = await player_repo.create("Player3", "NA1")
        
        # All players join the same game
        shared_game_id = "multiplayer_game_123"
        states = []
        for player in [player1, player2, player3]:
            state = await repo.create(
                player.id,
                "IN_GAME",
                game_id=shared_game_id,
                queue_type="RANKED_SOLO_5x5"
            )
            states.append(state)
        
        # Verify all players are tracked for the same game
        game_states = await repo.get_by_game_id(shared_game_id)
        assert len(game_states) == 3
        
        player_ids = {state.player_id for state in game_states}
        expected_player_ids = {player1.id, player2.id, player3.id}
        assert player_ids == expected_player_ids


# Import asyncio for sleep functionality
import asyncio