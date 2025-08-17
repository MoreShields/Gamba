"""End-to-end integration test for LoL Tracker service."""

import pytest
from tests.conftest import BaseE2ETest


@pytest.mark.asyncio
class TestLoLTrackerE2E(BaseE2ETest):
    """End-to-end test for the LoL Tracker happy path flow."""
    
    async def test_game_state_transitions_happy_path(
        self,
        lol_tracker_service,  # Needed to ensure polling is running
        grpc_client,
        mock_riot_control,
        mock_event_publisher,
        database_manager
    ):
        """Test the complete flow: track summoner -> start game -> end game with results."""
        # Setup clean test environment
        await self.setup_test_environment(mock_riot_control, mock_event_publisher, database_manager)
        
        # Test data
        game_name = "TestPlayer"
        tag_line = "NA1"
        puuid = "test-puuid-123"
        
        # Create and track player
        await self.create_test_player(mock_riot_control, game_name, tag_line, puuid)
        await self.track_summoner_via_grpc(grpc_client, game_name, tag_line, puuid)
        tracked_player = await self.verify_player_tracked_in_db(database_manager, game_name, tag_line)
        
        # Clear any events from initial setup/tracking
        mock_event_publisher.published_messages.clear()
        
        # Start a LoL game for the player
        game_data = await mock_riot_control.start_game(
            puuid=puuid,
            queue_type_id=420,  # Ranked Solo/Duo
            champion_id=157  # Yasuo
        )
        game_id = game_data["game_id"]
        
        # Wait for polling to detect game start (wait longer for slow test runs)
        await self.wait_for_polling_cycle(wait_time=3.0)
        
        # Verify game start event and database state
        game_start_events = self.find_game_state_events(mock_event_publisher)
        assert len(game_start_events) > 0, f"No game state events found. All events: {mock_event_publisher.published_messages}"
        
        start_event = game_start_events[-1]
        self.assert_game_start_event(start_event, game_id)
        await self.verify_game_state_in_db(database_manager, tracked_player, "IN_GAME", game_id)
        
        # End the LoL game with results
        await mock_riot_control.end_game(
            puuid=puuid,
            won=True,
            duration_seconds=2100,  # 35 minutes
            kills=12,
            deaths=4,
            assists=8
        )
        
        # Wait for polling to detect game end (completion loop runs every 1 second)
        await self.wait_for_polling_cycle(wait_time=3.0)
        
        # Verify game end event and LoL-specific results
        game_end_events = self.find_game_end_events(mock_event_publisher)
        assert len(game_end_events) > 0
        
        end_event = game_end_events[-1]
        self.assert_game_end_event(end_event)
        
        # Verify LoL-specific game result
        game_result = self.assert_game_result_present(end_event)
        assert game_result["won"] is True
        assert game_result["duration_seconds"] == 2100
        # Champion information should be present in some form
        assert (game_result.get("champion_played") is not None or 
                game_result.get("champion_name") is not None or
                game_result.get("champion_id") is not None)
        
        # Verify final database state
        await self.verify_game_state_in_db(database_manager, tracked_player, "NOT_IN_GAME", game_id)