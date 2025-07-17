"""Integration tests for TrackedPlayerRepository."""

import pytest
from datetime import datetime
from sqlalchemy.ext.asyncio import AsyncSession

from lol_tracker.database.repository import TrackedPlayerRepository
from lol_tracker.database.models import TrackedPlayer
from tests.factories import TrackedPlayerFactory
from tests.utils import count_tracked_players, assert_player_equals


@pytest.mark.integration
class TestTrackedPlayerRepository:
    """Test suite for TrackedPlayerRepository integration tests."""

    @pytest.mark.asyncio
    async def test_create_tracked_player(self, db_session: AsyncSession):
        """Test creating a new tracked player."""
        repo = TrackedPlayerRepository(db_session)

        player = await repo.create(
            game_name="TestPlayer",
            tag_line="gamba",
            puuid="test_puuid_123",
        )

        assert player.id is not None
        assert player.game_name == "TestPlayer"
        assert player.puuid == "test_puuid_123"
        assert player.is_active is True
        assert isinstance(player.created_at, datetime)
        assert isinstance(player.updated_at, datetime)

    @pytest.mark.asyncio
    async def test_create_tracked_player_minimal_data(self, db_session: AsyncSession):
        """Test creating a tracked player with required fields."""
        repo = TrackedPlayerRepository(db_session)

        player = await repo.create(
            game_name="MinimalPlayer",
            tag_line="gamba",
            puuid="test_puuid_minimal",
        )

        assert player.id is not None
        assert player.game_name == "MinimalPlayer"
        assert player.puuid == "test_puuid_minimal"
        assert player.is_active is True

    @pytest.mark.asyncio
    async def test_get_by_id(self, db_session: AsyncSession):
        """Test retrieving a tracked player by ID."""
        repo = TrackedPlayerRepository(db_session)

        # Create a player
        created_player = await repo.create("GetByIdTest", "gamba", "test_puuid_getbyid")
        player_id = created_player.id

        # Retrieve by ID
        retrieved_player = await repo.get_by_id(player_id)

        assert retrieved_player is not None
        assert_player_equals(retrieved_player, created_player, ignore_id=False)

    @pytest.mark.asyncio
    async def test_get_by_id_not_found(self, db_session: AsyncSession):
        """Test retrieving a non-existent player by ID."""
        repo = TrackedPlayerRepository(db_session)

        player = await repo.get_by_id(99999)

        assert player is None




    @pytest.mark.asyncio
    async def test_get_by_puuid(self, db_session: AsyncSession):
        """Test retrieving a tracked player by PUUID."""
        repo = TrackedPlayerRepository(db_session)

        # Create a player with PUUID
        created_player = await repo.create("PuuidTest", "gamba", puuid="unique_puuid_123")

        # Retrieve by PUUID
        retrieved_player = await repo.get_by_puuid("unique_puuid_123")

        assert retrieved_player is not None
        assert_player_equals(retrieved_player, created_player, ignore_id=False)

    @pytest.mark.asyncio
    async def test_get_by_puuid_not_found(self, db_session: AsyncSession):
        """Test retrieving a non-existent player by PUUID."""
        repo = TrackedPlayerRepository(db_session)

        player = await repo.get_by_puuid("non_existent_puuid")

        assert player is None

    @pytest.mark.asyncio
    async def test_get_all_active(self, db_session: AsyncSession):
        """Test retrieving all active tracked players."""
        repo = TrackedPlayerRepository(db_session)

        # Create multiple players
        player1 = await repo.create("Active1", "gamba", "test_puuid_active1")
        player2 = await repo.create("Active2", "gamba", "test_puuid_active2")
        player3 = await repo.create("Inactive", "gamba", "test_puuid_inactive")

        # Deactivate one player
        await repo.set_active_status(player3.id, False)

        # Get all active players
        active_players = await repo.get_all_active()

        assert len(active_players) == 2
        active_names = {p.game_name for p in active_players}
        assert active_names == {"Active1", "Active2"}

    @pytest.mark.asyncio
    async def test_get_all_active_empty(self, db_session: AsyncSession):
        """Test retrieving active players when none exist."""
        repo = TrackedPlayerRepository(db_session)

        active_players = await repo.get_all_active()

        assert active_players == []

    @pytest.mark.asyncio
    async def test_update_puuid(self, db_session: AsyncSession):
        """Test updating PUUID for a tracked player."""
        repo = TrackedPlayerRepository(db_session)

        # Create a player
        player = await repo.create("UpdateTest", "gamba", "old_puuid")
        original_updated_at = player.updated_at

        # Update PUUID
        success = await repo.update_puuid(
            player.id,
            puuid="new_puuid_123",
        )

        assert success is True

        # Verify the update
        updated_player = await repo.get_by_id(player.id)
        assert updated_player.puuid == "new_puuid_123"
        assert updated_player.updated_at > original_updated_at

    @pytest.mark.asyncio
    async def test_update_puuid_existing(self, db_session: AsyncSession):
        """Test updating PUUID for player with existing PUUID."""
        repo = TrackedPlayerRepository(db_session)

        # Create a player with PUUID
        player = await repo.create(
            "PartialTest", "gamba", puuid="existing_puuid"
        )

        # Update PUUID
        success = await repo.update_puuid(player.id, puuid="updated_puuid")

        assert success is True

        # Verify the update
        updated_player = await repo.get_by_id(player.id)
        assert updated_player.puuid == "updated_puuid"


    @pytest.mark.asyncio
    async def test_update_puuid_nonexistent_player(self, db_session: AsyncSession):
        """Test updating PUUID for a non-existent player."""
        repo = TrackedPlayerRepository(db_session)

        success = await repo.update_puuid(99999, puuid="some_puuid")

        assert success is False

    @pytest.mark.asyncio
    async def test_set_active_status(self, db_session: AsyncSession):
        """Test setting the active status of a tracked player."""
        repo = TrackedPlayerRepository(db_session)

        # Create an active player
        player = await repo.create("StatusTest", "gamba", "test_puuid_status")
        assert player.is_active is True

        # Deactivate the player
        success = await repo.set_active_status(player.id, False)
        assert success is True

        # Verify the status change
        updated_player = await repo.get_by_id(player.id)
        assert updated_player.is_active is False

        # Reactivate the player
        success = await repo.set_active_status(player.id, True)
        assert success is True

        # Verify the status change
        reactivated_player = await repo.get_by_id(player.id)
        assert reactivated_player.is_active is True

    @pytest.mark.asyncio
    async def test_set_active_status_nonexistent_player(self, db_session: AsyncSession):
        """Test setting active status for a non-existent player."""
        repo = TrackedPlayerRepository(db_session)

        success = await repo.set_active_status(99999, False)

        assert success is False

    @pytest.mark.asyncio
    async def test_delete_tracked_player(self, db_session: AsyncSession):
        """Test deleting a tracked player."""
        repo = TrackedPlayerRepository(db_session)

        # Create a player
        player = await repo.create("DeleteTest", "gamba", "test_puuid_delete")
        player_id = player.id

        # Verify player exists
        assert await repo.get_by_id(player_id) is not None

        # Delete the player
        success = await repo.delete(player_id)
        assert success is True

        # Verify player is deleted
        assert await repo.get_by_id(player_id) is None

    @pytest.mark.asyncio
    async def test_delete_nonexistent_player(self, db_session: AsyncSession):
        """Test deleting a non-existent player."""
        repo = TrackedPlayerRepository(db_session)

        success = await repo.delete(99999)

        assert success is False

    @pytest.mark.asyncio
    async def test_database_constraints(self, db_session: AsyncSession):
        """Test database constraints and indexes."""
        repo = TrackedPlayerRepository(db_session)

        # Create multiple players to test constraints
        players = TrackedPlayerFactory.create_multiple(5)

        for player_data in players:
            created_player = await repo.create(
                game_name=player_data.game_name,
                tag_line=player_data.tag_line,
                puuid=player_data.puuid,
            )
            assert created_player.id is not None

        # Verify all players were created
        assert await count_tracked_players(db_session) == 5

    @pytest.mark.asyncio
    async def test_concurrent_operations(self, db_session: AsyncSession):
        """Test that repository operations work correctly with database session."""
        repo = TrackedPlayerRepository(db_session)

        # Create a player
        player = await repo.create("ConcurrentTest", "gamba", "test_puuid_concurrent")

        # Perform multiple operations in the same session
        await repo.update_puuid(player.id, puuid="concurrent_puuid")
        await repo.set_active_status(player.id, False)

        # Verify all operations took effect
        final_player = await repo.get_by_id(player.id)
        assert final_player.puuid == "concurrent_puuid"
        assert final_player.is_active is False
