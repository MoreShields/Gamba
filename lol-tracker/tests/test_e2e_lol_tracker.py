"""End-to-end integration test for LoL Tracker service."""

import asyncio
import pytest
from lol_tracker.proto.services import summoner_service_pb2


@pytest.mark.asyncio
class TestLoLTrackerE2E:
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
        # Reset mock server state
        await mock_riot_control.reset_server()
        
        # Test data
        game_name = "TestPlayer"
        tag_line = "NA1"
        puuid = "test-puuid-123"
        
        # Create player in mock Riot API
        player_data = await mock_riot_control.create_player(game_name, tag_line, puuid)
        assert player_data["puuid"] == puuid
        
        # Start tracking the summoner via gRPC
        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name=game_name,
            tag_line=tag_line
        )
        response = await grpc_client.StartTrackingSummoner(request)
        assert response.success is True
        assert response.summoner_details.puuid == puuid
        
        # Verify player is in database
        tracked_player = await database_manager.get_tracked_player_by_puuid(puuid)
        assert tracked_player is not None
        assert tracked_player.game_name == game_name
        assert tracked_player.tag_line == tag_line
        
        # Start a game for the player
        game_data = await mock_riot_control.start_game(
            puuid=puuid,
            queue_type_id=420,  # Ranked Solo/Duo
            champion_id=157  # Yasuo
        )
        game_id = game_data["game_id"]
        
        # Wait for automatic polling to detect game start (polling interval is 1 second)
        await asyncio.sleep(2.0)  # Wait for at least one polling cycle
        
        # Verify game start event was published
        events = mock_event_publisher.published_messages
        game_start_events = [e for e in events if "state_changed" in e.get("subject", "")]
        assert len(game_start_events) > 0, f"No game state events found. All events: {events}"
        
        # Check the game start event details
        start_event = game_start_events[-1]
        assert start_event["previous_status"] == "NOT_IN_GAME"
        assert start_event["new_status"] == "IN_GAME"
        assert start_event["game_id"] == game_id
        assert start_event["is_game_start"] is True
        assert start_event["is_game_end"] is False
        
        # Verify game state in database
        game_state = await database_manager.get_latest_game_state_for_player(tracked_player.id)
        assert game_state is not None
        assert game_state.status.value == "IN_GAME"
        assert game_state.game_id == game_id
        
        # End the game with results
        await mock_riot_control.end_game(
            puuid=puuid,
            won=True,
            duration_seconds=2100,  # 35 minutes
            kills=12,
            deaths=4,
            assists=8
        )
        
        # Wait for automatic polling to detect game end
        await asyncio.sleep(2.0)  # Wait for at least one polling cycle
        
        # Verify game end event was published with results
        all_events = mock_event_publisher.published_messages
        game_end_events = [e for e in all_events if e.get("is_game_end") is True]
        assert len(game_end_events) > 0
        
        # Check the game end event details
        end_event = game_end_events[-1]
        assert end_event["previous_status"] == "IN_GAME"
        assert end_event["new_status"] == "NOT_IN_GAME"
        assert end_event["is_game_start"] is False
        assert end_event["is_game_end"] is True
        
        # Verify game result is included
        assert "game_result" in end_event
        game_result = end_event["game_result"]
        assert game_result["won"] is True
        assert game_result["duration_seconds"] == 2100
        # Champion information should be present in some form
        assert (game_result.get("champion_played") is not None or 
                game_result.get("champion_name") is not None or
                game_result.get("champion_id") is not None)
        
        # Verify game state is back to NOT_IN_GAME in database
        updated_game_state = await database_manager.get_latest_game_state_for_player(tracked_player.id)
        assert updated_game_state.status.value == "NOT_IN_GAME"
        # Game ID is preserved for the last game that was played
        assert updated_game_state.game_id == game_id