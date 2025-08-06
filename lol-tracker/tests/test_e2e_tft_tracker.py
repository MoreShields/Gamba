"""End-to-end integration test for TFT Tracker service."""

import pytest
from tests.conftest import BaseE2ETest


@pytest.mark.asyncio
class TestTFTTrackerE2E(BaseE2ETest):
    """End-to-end test for the TFT Tracker happy path flow."""
    
    async def test_game_state_transitions_happy_path(
        self,
        lol_tracker_service,  # Needed to ensure polling is running
        grpc_client,
        mock_riot_control,
        mock_event_publisher,
        database_manager
    ):
        """Test the complete TFT flow: track summoner -> start game -> end game with placement results."""
        # Setup clean test environment
        await self.setup_test_environment(mock_riot_control, mock_event_publisher, database_manager)
        
        # Test data
        game_name = "TFTPlayer"
        tag_line = "NA1"
        puuid = "tft-puuid-456"
        
        # Create and track player
        await self.create_test_player(mock_riot_control, game_name, tag_line, puuid)
        await self.track_summoner_via_grpc(grpc_client, game_name, tag_line, puuid)
        tracked_player = await self.verify_player_tracked_in_db(database_manager, puuid, game_name, tag_line)
        
        # Start a TFT game for the player
        game_data = await mock_riot_control.start_tft_game(
            puuid=puuid,
            queue_type_id=1100  # TFT Ranked
        )
        game_id = game_data["game_id"]
        
        # Wait for polling to detect game start
        await self.wait_for_polling_cycle()
        
        # Verify game start event and database state
        game_start_events = self.find_game_state_events(mock_event_publisher)
        assert len(game_start_events) > 0, f"No game state events found. All events: {mock_event_publisher.published_messages}"
        
        start_event = game_start_events[-1]
        self.assert_game_start_event(start_event, game_id)
        await self.verify_game_state_in_db(database_manager, tracked_player, "IN_GAME", game_id)
        
        # End the TFT game with a 2nd place finish (win)
        await mock_riot_control.end_tft_game(
            puuid=puuid,
            placement=2,  # 2nd place (top 4 is a win)
            duration_seconds=2400  # 40 minutes
        )
        
        # Wait for polling to detect game end
        await self.wait_for_polling_cycle()
        
        # Verify game end event and TFT-specific results
        game_end_events = self.find_game_end_events(mock_event_publisher)
        assert len(game_end_events) > 0
        
        end_event = game_end_events[-1]
        self.assert_game_end_event(end_event)
        
        # Verify TFT-specific game result
        game_result = self.assert_game_result_present(end_event)
        assert game_result["placement"] == 2
        assert game_result["duration_seconds"] == 2400
        
        # Verify final database state
        await self.verify_game_state_in_db(database_manager, tracked_player, "NOT_IN_GAME", game_id)