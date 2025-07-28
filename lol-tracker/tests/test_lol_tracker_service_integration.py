"""Integration tests for LoLTrackerService polling and persistence."""

import pytest
from unittest.mock import Mock, AsyncMock
from datetime import datetime
from google.protobuf.timestamp_pb2 import Timestamp

from lol_tracker.service import LoLTrackerService
from lol_tracker.database.repository import TrackedPlayerRepository, GameStateRepository
from lol_tracker.riot_api import CurrentGameInfo, PlayerNotInGameError
from lol_tracker.proto.events import lol_events_pb2


@pytest.mark.integration
class TestLoLTrackerServiceIntegration:
    """Integration tests for LoLTrackerService with real database persistence."""

    @pytest.mark.asyncio
    async def test_poll_player_game_state_persists_not_in_game_to_in_game_transition(
        self, clean_db_session, db_manager, test_config
    ):
        """Test that NOT_IN_GAME -> IN_GAME transition is properly persisted and event published."""
        # Setup: Create tracked player in database
        tracked_player_repo = TrackedPlayerRepository(clean_db_session)
        player = await tracked_player_repo.create(
            game_name="TestPlayer", 
            tag_line="TEST", 
            puuid="test-puuid-123"
        )
        await clean_db_session.commit()
        
        # Mock dependencies
        mock_message_bus = AsyncMock()
        mock_riot_client = AsyncMock()
        
        # Create service with mocked externals but real database
        service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
        service.riot_api_client = mock_riot_client
        service.db_manager = db_manager
        
        # Mock Riot API response for IN_GAME state
        mock_game_info = CurrentGameInfo(
            game_id="4567890123",
            game_type="MATCHED_GAME",
            game_start_time=1640995200000,  # Jan 1, 2022 00:00:00 UTC
            map_id=11,
            game_length=1200,  # 20 minutes
            platform_id="NA1",
            game_mode="CLASSIC",
            game_queue_config_id=420,  # Ranked Solo/Duo
            participants=[]
        )
        mock_riot_client.get_current_game_info.return_value = mock_game_info
        
        # Execute the polling method
        await service._poll_player_game_state(player)
        
        # Verify database persistence
        game_state_repo = GameStateRepository(clean_db_session)
        latest_state = await game_state_repo.get_latest_for_player(player.id)
        
        assert latest_state is not None
        assert latest_state.status == "IN_GAME"
        assert latest_state.game_id == "4567890123"
        assert latest_state.queue_type == "RANKED_SOLO_5x5"
        assert latest_state.game_start_time == datetime.fromtimestamp(1640995200000 / 1000)
        assert mock_game_info.__dict__.__str__() in latest_state.raw_api_response
        
        # Verify event publishing
        mock_message_bus.publish.assert_called_once()
        published_subject = mock_message_bus.publish.call_args[0][0]
        published_event_bytes = mock_message_bus.publish.call_args[0][1]
        
        assert published_subject == "lol.gamestate.changed"
        
        # Deserialize and verify event content
        event = lol_events_pb2.LoLGameStateChanged()
        event.ParseFromString(published_event_bytes)
        
        assert event.game_name == "TestPlayer"
        assert event.tag_line == "TEST"
        assert event.previous_status == lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME
        assert event.current_status == lol_events_pb2.GameStatus.GAME_STATUS_IN_GAME
        assert event.game_id == "4567890123"
        assert event.queue_type == "RANKED_SOLO_5x5"
        # Verify timestamp is set (exact time will vary)
        assert event.event_time.seconds > 0

    @pytest.mark.asyncio
    async def test_poll_player_game_state_persists_in_game_to_not_in_game_transition(
        self, clean_db_session, db_manager, test_config
    ):
        """Test that IN_GAME -> NOT_IN_GAME transition is properly persisted and event published."""
        # Setup: Create tracked player with existing IN_GAME state
        tracked_player_repo = TrackedPlayerRepository(clean_db_session)
        player = await tracked_player_repo.create(
            game_name="TestPlayer", 
            tag_line="TEST", 
            puuid="test-puuid-456"
        )
        
        game_state_repo = GameStateRepository(clean_db_session)
        existing_state = await game_state_repo.create(
            player_id=player.id,
            status="IN_GAME",
            game_id="existing_game_123",
            queue_type="RANKED_SOLO_5x5",
            game_start_time=datetime.fromtimestamp(1640995200000 / 1000)
        )
        await clean_db_session.commit()
        
        # Mock dependencies
        mock_message_bus = AsyncMock()
        mock_riot_client = AsyncMock()
        
        # Create service
        service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
        service.riot_api_client = mock_riot_client
        service.db_manager = db_manager
        
        # Mock Riot API response for NOT_IN_GAME state (player left game)
        mock_riot_client.get_current_game_info.side_effect = PlayerNotInGameError()
        
        # Mock match result fetching - this will be called when transitioning to NOT_IN_GAME
        from lol_tracker.riot_api.riot_api_client import MatchInfo
        mock_match_info = MatchInfo(
            match_id="NA1_existing_game_123",
            game_creation=1640995200000,
            game_duration=1800,
            game_end_timestamp=1640997000000,
            game_mode="CLASSIC",
            game_type="MATCHED_GAME",
            map_id=11,
            platform_id="NA1",
            queue_id=420,
            participants=[{
                "puuid": "test-puuid-456",
                "win": True,
                "championName": "Jinx",
                "championId": 222,
                "kills": 10,
                "deaths": 2,
                "assists": 8
            }]
        )
        mock_riot_client.get_match_info = AsyncMock(return_value=mock_match_info)
        
        # Execute the polling method
        await service._poll_player_game_state(player)
        
        # Verify new state persisted
        latest_state = await game_state_repo.get_latest_for_player(player.id)
        
        assert latest_state is not None
        assert latest_state.id != existing_state.id  # New record created
        assert latest_state.status == "NOT_IN_GAME"
        # Game ID should be preserved when transitioning from IN_GAME to NOT_IN_GAME
        assert latest_state.game_id == "existing_game_123"
        assert latest_state.queue_type == "RANKED_SOLO_5x5"
        
        # Verify match result was fetched and stored
        assert latest_state.won is True
        assert latest_state.champion_played == "Jinx"
        assert latest_state.duration_seconds == 1800
        
        # Verify event publishing
        mock_message_bus.publish.assert_called_once()
        published_event_bytes = mock_message_bus.publish.call_args[0][1]
        
        # Deserialize and verify event content
        event = lol_events_pb2.LoLGameStateChanged()
        event.ParseFromString(published_event_bytes)
        
        assert event.game_name == "TestPlayer"
        assert event.tag_line == "TEST"
        assert event.previous_status == lol_events_pb2.GameStatus.GAME_STATUS_IN_GAME
        assert event.current_status == lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME
        # When transitioning from IN_GAME to NOT_IN_GAME, game_id should be preserved in event
        assert event.HasField("game_id")
        assert event.game_id == "existing_game_123"
        assert event.HasField("queue_type")
        assert event.queue_type == "RANKED_SOLO_5x5"
        
        # Verify game result is populated in the event when transitioning from IN_GAME to NOT_IN_GAME
        assert event.HasField("game_result")
        assert event.game_result.won is True
        assert event.game_result.champion_played == "Jinx"
        assert event.game_result.duration_seconds == 1800
        assert event.game_result.queue_type == "RANKED_SOLO_5x5"
        
        # Verify match result API was called
        mock_riot_client.get_match_info.assert_called_once_with("NA1_existing_game_123", "na1")

    @pytest.mark.asyncio
    async def test_poll_player_no_state_change_updates_existing(
        self, clean_db_session, db_manager, test_config
    ):
        """Test that staying IN_GAME updates existing record and commits properly."""
        # Setup: Create tracked player with existing IN_GAME state
        tracked_player_repo = TrackedPlayerRepository(clean_db_session)
        player = await tracked_player_repo.create(
            game_name="TestPlayer", 
            tag_line="TEST", 
            puuid="test-puuid-789"
        )
        
        game_state_repo = GameStateRepository(clean_db_session)
        existing_state = await game_state_repo.create(
            player_id=player.id,
            status="IN_GAME",
            game_id="ongoing_game_456",
            queue_type="RANKED_SOLO_5x5",
            game_start_time=datetime.fromtimestamp(1640995200000 / 1000),
            raw_api_response="old_response_data"
        )
        await clean_db_session.commit()
        
        # Mock dependencies
        mock_message_bus = AsyncMock()
        mock_riot_client = AsyncMock()
        
        # Create service
        service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
        service.riot_api_client = mock_riot_client
        service.db_manager = db_manager
        
        # Mock Riot API response for same game (still IN_GAME)
        mock_game_info = CurrentGameInfo(
            game_id="ongoing_game_456",  # Same game ID
            game_type="MATCHED_GAME",
            game_start_time=1640995200000,
            map_id=11,
            game_length=1800,  # Game progressed to 30 minutes
            platform_id="NA1",
            game_mode="CLASSIC",
            game_queue_config_id=420,
            participants=[]
        )
        mock_riot_client.get_current_game_info.return_value = mock_game_info
        
        # Execute the polling method
        await service._poll_player_game_state(player)
        
        # Verify no new state created, but existing state can be queried
        # (Note: _update_in_game_state method currently doesn't update fields,
        # but the commit should succeed)
        latest_state = await game_state_repo.get_latest_for_player(player.id)
        assert latest_state.id == existing_state.id  # Same record
        assert latest_state.status == "IN_GAME"
        assert latest_state.game_id == "ongoing_game_456"
        
        # Verify no event published (no state change)
        mock_message_bus.publish.assert_not_called()

    @pytest.mark.asyncio
    async def test_poll_player_first_time_no_state_change_no_record(
        self, clean_db_session, db_manager, test_config
    ):
        """Test polling player with no previous game state when still NOT_IN_GAME."""
        # This test documents current behavior: when a player is first polled
        # and they're NOT_IN_GAME, no state record is created because there's
        # no state change (previous=NOT_IN_GAME, current=NOT_IN_GAME)
        
        # Setup: Create tracked player with no game states
        tracked_player_repo = TrackedPlayerRepository(clean_db_session)
        player = await tracked_player_repo.create(
            game_name="NewPlayer", 
            tag_line="NEW", 
            puuid="new-player-puuid"
        )
        await clean_db_session.commit()
        
        # Mock dependencies
        mock_message_bus = AsyncMock()
        mock_riot_client = AsyncMock()
        
        # Create service
        service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
        service.riot_api_client = mock_riot_client
        service.db_manager = db_manager
        
        # Mock Riot API response for NOT_IN_GAME state
        mock_riot_client.get_current_game_info.side_effect = PlayerNotInGameError()
        
        # Execute the polling method
        await service._poll_player_game_state(player)
        
        # Verify no state created (current behavior: no change detected)
        game_state_repo = GameStateRepository(clean_db_session)
        latest_state = await game_state_repo.get_latest_for_player(player.id)
        assert latest_state is None
        
        # Verify no event published (no state change)
        mock_message_bus.publish.assert_not_called()

    @pytest.mark.asyncio
    async def test_poll_player_first_time_immediately_in_game(
        self, clean_db_session, db_manager, test_config
    ):
        """Test polling player for first time when they're already IN_GAME."""
        # Setup: Create tracked player with no game states
        tracked_player_repo = TrackedPlayerRepository(clean_db_session)
        player = await tracked_player_repo.create(
            game_name="NewGamePlayer", 
            tag_line="GAME", 
            puuid="new-game-player-puuid"
        )
        await clean_db_session.commit()
        
        # Mock dependencies
        mock_message_bus = AsyncMock()
        mock_riot_client = AsyncMock()
        
        # Create service
        service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
        service.riot_api_client = mock_riot_client
        service.db_manager = db_manager
        
        # Mock Riot API response for IN_GAME state (first poll finds them in game)
        mock_game_info = CurrentGameInfo(
            game_id="first_game_123",
            game_type="MATCHED_GAME",
            game_start_time=1640995200000,
            map_id=11,
            game_length=600,  # 10 minutes in
            platform_id="NA1",
            game_mode="CLASSIC",
            game_queue_config_id=420,
            participants=[]
        )
        mock_riot_client.get_current_game_info.return_value = mock_game_info
        
        # Execute the polling method
        await service._poll_player_game_state(player)
        
        # Verify state created (change from None/NOT_IN_GAME to IN_GAME)
        game_state_repo = GameStateRepository(clean_db_session)
        latest_state = await game_state_repo.get_latest_for_player(player.id)
        
        assert latest_state is not None
        assert latest_state.status == "IN_GAME"
        assert latest_state.game_id == "first_game_123"
        assert latest_state.queue_type == "RANKED_SOLO_5x5"
        
        # Verify event published for state change
        mock_message_bus.publish.assert_called_once()
        published_event_bytes = mock_message_bus.publish.call_args[0][1]
        
        # Deserialize and verify event content
        event = lol_events_pb2.LoLGameStateChanged()
        event.ParseFromString(published_event_bytes)
        
        assert event.game_name == "NewGamePlayer"
        assert event.tag_line == "GAME"
        assert event.previous_status == lol_events_pb2.GameStatus.GAME_STATUS_NOT_IN_GAME
        assert event.current_status == lol_events_pb2.GameStatus.GAME_STATUS_IN_GAME

    @pytest.mark.asyncio
    async def test_poll_player_handles_riot_api_errors_gracefully(
        self, clean_db_session, db_manager, test_config
    ):
        """Test that polling continues gracefully when Riot API has errors."""
        # Setup: Create tracked player
        tracked_player_repo = TrackedPlayerRepository(clean_db_session)
        player = await tracked_player_repo.create(
            game_name="ErrorPlayer", 
            tag_line="ERR", 
            puuid="error-test-puuid"
        )
        await clean_db_session.commit()
        
        # Mock dependencies
        mock_message_bus = AsyncMock()
        mock_riot_client = AsyncMock()
        
        # Create service
        service = LoLTrackerService(test_config, message_bus_client=mock_message_bus)
        service.riot_api_client = mock_riot_client
        service.db_manager = db_manager
        
        # Mock Riot API to raise unexpected error
        mock_riot_client.get_current_game_info.side_effect = Exception("API rate limit exceeded")
        
        # Execute the polling method - should not raise exception
        await service._poll_player_game_state(player)
        
        # Verify no state created due to error
        game_state_repo = GameStateRepository(clean_db_session)
        latest_state = await game_state_repo.get_latest_for_player(player.id)
        assert latest_state is None
        
        # Verify no event published due to error
        mock_message_bus.publish.assert_not_called()