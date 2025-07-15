"""Tests for protobuf model instantiation and serialization."""
from datetime import datetime

import pytest
from google.protobuf.timestamp_pb2 import Timestamp

from lol_tracker.proto import (
    GameStatus,
    LoLGameStateChanged,
    GameResult,
    PlayerTrackingCommand,
    StartTrackingPlayer,
    StopTrackingPlayer,
)


class TestProtoInstantiation:
    """Test that protobuf models can be instantiated correctly."""

    def test_game_status_enum(self):
        """Test GameStatus enum values."""
        assert GameStatus.GAME_STATUS_UNKNOWN == 0
        assert GameStatus.GAME_STATUS_NOT_IN_GAME == 1
        assert GameStatus.GAME_STATUS_IN_CHAMPION_SELECT == 2
        assert GameStatus.GAME_STATUS_IN_GAME == 3

    def test_lol_game_state_changed_creation(self):
        """Test creating LoLGameStateChanged message."""
        # Create timestamp
        now = Timestamp()
        now.GetCurrentTime()
        
        # Create game result
        game_result = GameResult(
            won=True,
            duration_seconds=1800,  # 30 minutes
            queue_type="RANKED_SOLO_5x5",
            champion_played="Jinx"
        )
        
        # Create main event
        event = LoLGameStateChanged(
            summoner_name="TestSummoner",
            region="na1",
            previous_status=GameStatus.GAME_STATUS_IN_GAME,
            current_status=GameStatus.GAME_STATUS_NOT_IN_GAME,
            game_result=game_result,
            event_time=now,
            game_id="NA1_1234567890",
            queue_type="RANKED_SOLO_5x5"
        )
        
        # Verify fields
        assert event.summoner_name == "TestSummoner"
        assert event.region == "na1"
        assert event.previous_status == GameStatus.GAME_STATUS_IN_GAME
        assert event.current_status == GameStatus.GAME_STATUS_NOT_IN_GAME
        assert event.game_result.won is True
        assert event.game_result.duration_seconds == 1800
        assert event.game_result.champion_played == "Jinx"
        assert event.game_id == "NA1_1234567890"

    def test_start_tracking_player_creation(self):
        """Test creating StartTrackingPlayer message."""
        now = Timestamp()
        now.GetCurrentTime()
        
        start_tracking = StartTrackingPlayer(
            summoner_name="PlayerToTrack",
            region="na1",
            requested_at=now
        )
        
        assert start_tracking.summoner_name == "PlayerToTrack"
        assert start_tracking.region == "na1"
        assert start_tracking.requested_at == now

    def test_stop_tracking_player_creation(self):
        """Test creating StopTrackingPlayer message."""
        now = Timestamp()
        now.GetCurrentTime()
        
        stop_tracking = StopTrackingPlayer(
            summoner_name="PlayerToStop",
            region="euw1",
            requested_at=now
        )
        
        assert stop_tracking.summoner_name == "PlayerToStop"
        assert stop_tracking.region == "euw1"
        assert stop_tracking.requested_at == now

    def test_player_tracking_command_with_start(self):
        """Test PlayerTrackingCommand with start tracking."""
        now = Timestamp()
        now.GetCurrentTime()
        
        start_tracking = StartTrackingPlayer(
            summoner_name="NewPlayer",
            region="na1",
            requested_at=now
        )
        
        command = PlayerTrackingCommand(start_tracking=start_tracking)
        
        assert command.HasField("start_tracking")
        assert not command.HasField("stop_tracking")
        assert command.start_tracking.summoner_name == "NewPlayer"

    def test_player_tracking_command_with_stop(self):
        """Test PlayerTrackingCommand with stop tracking."""
        now = Timestamp()
        now.GetCurrentTime()
        
        stop_tracking = StopTrackingPlayer(
            summoner_name="OldPlayer",
            region="euw1",
            requested_at=now
        )
        
        command = PlayerTrackingCommand(stop_tracking=stop_tracking)
        
        assert command.HasField("stop_tracking")
        assert not command.HasField("start_tracking")
        assert command.stop_tracking.summoner_name == "OldPlayer"


class TestProtoSerialization:
    """Test protobuf serialization and deserialization."""

    def test_lol_game_state_changed_serialization(self):
        """Test serializing and deserializing LoLGameStateChanged."""
        # Create original event
        now = Timestamp()
        now.GetCurrentTime()
        
        game_result = GameResult(
            won=False,
            duration_seconds=2400,
            queue_type="NORMAL",
            champion_played="Yasuo"
        )
        
        original_event = LoLGameStateChanged(
            summoner_name="SerializationTest",
            region="kr",
            previous_status=GameStatus.GAME_STATUS_NOT_IN_GAME,
            current_status=GameStatus.GAME_STATUS_IN_GAME,
            game_result=game_result,
            event_time=now,
            game_id="KR_9876543210"
        )
        
        # Serialize
        serialized_data = original_event.SerializeToString()
        assert isinstance(serialized_data, bytes)
        assert len(serialized_data) > 0
        
        # Deserialize
        deserialized_event = LoLGameStateChanged()
        deserialized_event.ParseFromString(serialized_data)
        
        # Verify all fields match
        assert deserialized_event.summoner_name == original_event.summoner_name
        assert deserialized_event.region == original_event.region
        assert deserialized_event.previous_status == original_event.previous_status
        assert deserialized_event.current_status == original_event.current_status
        assert deserialized_event.game_result.won == original_event.game_result.won
        assert deserialized_event.game_result.duration_seconds == original_event.game_result.duration_seconds
        assert deserialized_event.game_result.champion_played == original_event.game_result.champion_played
        assert deserialized_event.game_id == original_event.game_id

    def test_player_tracking_command_serialization(self):
        """Test serializing and deserializing PlayerTrackingCommand."""
        now = Timestamp()
        now.GetCurrentTime()
        
        start_tracking = StartTrackingPlayer(
            summoner_name="SerializeMe",
            region="na1",
            requested_at=now
        )
        
        original_command = PlayerTrackingCommand(start_tracking=start_tracking)
        
        # Serialize
        serialized_data = original_command.SerializeToString()
        assert isinstance(serialized_data, bytes)
        assert len(serialized_data) > 0
        
        # Deserialize
        deserialized_command = PlayerTrackingCommand()
        deserialized_command.ParseFromString(serialized_data)
        
        # Verify
        assert deserialized_command.HasField("start_tracking")
        assert deserialized_command.start_tracking.summoner_name == "SerializeMe"
        assert deserialized_command.start_tracking.region == "na1"