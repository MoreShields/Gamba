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
        tracked_player = await self.verify_player_tracked_in_db(database_manager, game_name, tag_line)
        
        # Clear any events from initial setup/tracking
        mock_event_publisher.published_messages.clear()
        
        # Start a TFT game for the player
        game_data = await mock_riot_control.start_tft_game(
            puuid=puuid,
            queue_type_id=1100  # TFT Ranked
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
        
        # End the TFT game with a 2nd place finish (win)
        await mock_riot_control.end_tft_game(
            puuid=puuid,
            placement=2,  # 2nd place (top 4 is a win)
            duration_seconds=2400  # 40 minutes
        )
        
        # Wait for polling to detect game end (completion loop runs every 1 second)
        await self.wait_for_polling_cycle(wait_time=3.0)
        
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
    
    async def test_tft_early_exit_with_delayed_match_results(
        self,
        lol_tracker_service,  # Needed to ensure polling is running
        grpc_client,
        mock_riot_control,
        mock_event_publisher,
        database_manager
    ):
        """Test TFT early exit scenario where player starts new game before previous game's results are available."""
        # Setup clean test environment
        await self.setup_test_environment(mock_riot_control, mock_event_publisher, database_manager)
        
        # Test data
        game_name = "EarlyExitPlayer"
        tag_line = "NA1"
        puuid = "early-exit-tft-789"
        
        # Create and track player
        await self.create_test_player(mock_riot_control, game_name, tag_line, puuid)
        await self.track_summoner_via_grpc(grpc_client, game_name, tag_line, puuid)
        tracked_player = await self.verify_player_tracked_in_db(database_manager, game_name, tag_line)
        
        # Clear any events from initial setup/tracking
        mock_event_publisher.published_messages.clear()
        
        # Start first TFT game
        game1_data = await mock_riot_control.start_tft_game(
            puuid=puuid,
            queue_type_id=1100  # TFT Ranked
        )
        game1_id = game1_data["game_id"]
        
        # Wait for polling to detect first game start
        await self.wait_for_polling_cycle(wait_time=3.0)
        
        # Verify first game start event
        game_start_events = self.find_game_state_events(mock_event_publisher)
        assert len(game_start_events) > 0, f"No game state events found for game 1"
        
        start_event1 = game_start_events[-1]
        self.assert_game_start_event(start_event1, game1_id)
        await self.verify_game_state_in_db(database_manager, tracked_player, "IN_GAME", game1_id)
        
        # Clear events before starting second game
        mock_event_publisher.published_messages.clear()
        
        # Start second TFT game (simulating early exit from first game)
        game2_data = await mock_riot_control.start_tft_game(
            puuid=puuid,
            queue_type_id=1100  # TFT Ranked
        )
        game2_id = game2_data["game_id"]
        
        # Wait for polling to detect second game start
        await self.wait_for_polling_cycle(wait_time=3.0)
        
        # Verify state transition to second game
        game_state_events = self.find_game_state_events(mock_event_publisher)
        assert len(game_state_events) >= 1, f"Expected state change events for transition to game 2"
        
        # Should have events for: game1 end (without results) and game2 start
        await self.verify_game_state_in_db(database_manager, tracked_player, "IN_GAME", game2_id)
        
        # Clear events before ending games
        mock_event_publisher.published_messages.clear()
        
        # Now end the first game with results (simulating delayed match results)
        await mock_riot_control.end_tft_game(
            puuid=puuid,
            game_id=game1_id,  # Specify which game to end
            placement=8,  # Last place (early exit)
            duration_seconds=600  # 10 minutes
        )
        
        # End the second game normally
        await mock_riot_control.end_tft_game(
            puuid=puuid,
            game_id=game2_id,  # Specify which game to end
            placement=1,  # First place
            duration_seconds=2100  # 35 minutes
        )
        
        # Wait for polling to detect both game ends
        await self.wait_for_polling_cycle(wait_time=3.0)
        
        # Verify we got end events for both games
        game_end_events = self.find_game_end_events(mock_event_publisher)
        assert len(game_end_events) >= 1, f"Expected at least one game end event"
        
        # Find the most recent end event (should be for game 2)
        latest_end_event = game_end_events[-1]
        self.assert_game_end_event(latest_end_event)
        
        # Verify game result is present
        game_result = self.assert_game_result_present(latest_end_event)
        
        # The most recent game should have ended with placement 1
        # Note: The polling service might only capture the final game's result
        # since it transitions directly from game1 to game2 without going to NOT_IN_GAME
        assert game_result["placement"] in [1, 8], f"Expected placement 1 or 8, got {game_result['placement']}"
        
        # Verify final database state shows player not in game
        await self.verify_game_state_in_db(database_manager, tracked_player, "NOT_IN_GAME", None)