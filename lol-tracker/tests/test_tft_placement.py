"""Tests for TFT Double Up placement calculation."""

import pytest
from lol_tracker.adapters.riot_api.client import TFTMatchInfo


class TestTFTDoubleUpPlacement:
    """Test suite for TFT Double Up placement adjustments."""
    
    def test_is_double_up_queue_normal(self):
        """Test identifying Normal Double Up queue."""
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=[],
            queue_id=1140,  # TFT Normal Double Up
            tft_game_type="standard",
            tft_set_number=10
        )
        assert match_info.is_double_up_queue() is True
    
    def test_is_double_up_queue_ranked(self):
        """Test identifying Ranked Double Up queue."""
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=[],
            queue_id=1160,  # TFT Ranked Double Up
            tft_game_type="standard",
            tft_set_number=10
        )
        assert match_info.is_double_up_queue() is True
    
    def test_is_double_up_queue_beta(self):
        """Test identifying Beta/Workshop Double Up queue."""
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=[],
            queue_id=1150,  # TFT Double Up (Beta/Workshop)
            tft_game_type="standard",
            tft_set_number=10
        )
        assert match_info.is_double_up_queue() is True
    
    def test_is_not_double_up_queue(self):
        """Test regular TFT queues are not identified as Double Up."""
        # Normal TFT
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=[],
            queue_id=1090,  # TFT Normal
            tft_game_type="standard",
            tft_set_number=10
        )
        assert match_info.is_double_up_queue() is False
        
        # Ranked TFT
        match_info.queue_id = 1100  # TFT Ranked
        assert match_info.is_double_up_queue() is False
        
        # Hyper Roll
        match_info.queue_id = 1130  # TFT Hyper Roll
        assert match_info.is_double_up_queue() is False
    
    def test_double_up_placement_adjustment(self):
        """Test placement adjustment for Double Up games."""
        participants = [
            {"riotIdGameName": "Player1", "riotIdTagline": "NA1", "placement": 1},
            {"riotIdGameName": "Player2", "riotIdTagline": "NA2", "placement": 2},
            {"riotIdGameName": "Player3", "riotIdTagline": "NA3", "placement": 3},
            {"riotIdGameName": "Player4", "riotIdTagline": "NA4", "placement": 4},
            {"riotIdGameName": "Player5", "riotIdTagline": "NA5", "placement": 5},
            {"riotIdGameName": "Player6", "riotIdTagline": "NA6", "placement": 6},
            {"riotIdGameName": "Player7", "riotIdTagline": "NA7", "placement": 7},
            {"riotIdGameName": "Player8", "riotIdTagline": "NA8", "placement": 8},
        ]
        
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=participants,
            queue_id=1160,  # Ranked Double Up
            tft_game_type="standard",
            tft_set_number=10
        )
        
        # Test placement mapping for Double Up
        # 1-2 -> 1, 3-4 -> 2, 5-6 -> 3, 7-8 -> 4
        assert match_info.get_placement_by_name("Player1", "NA1") == 1
        assert match_info.get_placement_by_name("Player2", "NA2") == 1
        assert match_info.get_placement_by_name("Player3", "NA3") == 2
        assert match_info.get_placement_by_name("Player4", "NA4") == 2
        assert match_info.get_placement_by_name("Player5", "NA5") == 3
        assert match_info.get_placement_by_name("Player6", "NA6") == 3
        assert match_info.get_placement_by_name("Player7", "NA7") == 4
        assert match_info.get_placement_by_name("Player8", "NA8") == 4
    
    def test_regular_tft_placement_unchanged(self):
        """Test that regular TFT games return unadjusted placement."""
        participants = [
            {"riotIdGameName": "Player1", "riotIdTagline": "NA1", "placement": 1},
            {"riotIdGameName": "Player2", "riotIdTagline": "NA2", "placement": 2},
            {"riotIdGameName": "Player3", "riotIdTagline": "NA3", "placement": 3},
            {"riotIdGameName": "Player4", "riotIdTagline": "NA4", "placement": 4},
            {"riotIdGameName": "Player5", "riotIdTagline": "NA5", "placement": 5},
            {"riotIdGameName": "Player6", "riotIdTagline": "NA6", "placement": 6},
            {"riotIdGameName": "Player7", "riotIdTagline": "NA7", "placement": 7},
            {"riotIdGameName": "Player8", "riotIdTagline": "NA8", "placement": 8},
        ]
        
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=participants,
            queue_id=1100,  # Regular Ranked TFT
            tft_game_type="standard",
            tft_set_number=10
        )
        
        # Test that regular TFT returns 1-8 placement unchanged
        for i in range(1, 9):
            assert match_info.get_placement_by_name(f"Player{i}", f"NA{i}") == i
    
    def test_get_placement_case_insensitive(self):
        """Test that get_placement_by_name is case insensitive."""
        participants = [
            {"riotIdGameName": "TestPlayer", "riotIdTagline": "NA1", "placement": 3},
        ]
        
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=participants,
            queue_id=1160,  # Double Up
            tft_game_type="standard",
            tft_set_number=10
        )
        
        # Should match regardless of case
        assert match_info.get_placement_by_name("testplayer", "na1") == 2  # 3 -> 2 in Double Up
        assert match_info.get_placement_by_name("TESTPLAYER", "NA1") == 2
        assert match_info.get_placement_by_name("TestPlayer", "na1") == 2
    
    def test_get_placement_player_not_found(self):
        """Test that get_placement_by_name returns None for unknown players."""
        participants = [
            {"riotIdGameName": "Player1", "riotIdTagline": "NA1", "placement": 1},
        ]
        
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=participants,
            queue_id=1160,  # Double Up
            tft_game_type="standard",
            tft_set_number=10
        )
        
        assert match_info.get_placement_by_name("UnknownPlayer", "NA1") is None
        assert match_info.get_placement_by_name("Player1", "WrongTag") is None
    
    def test_get_placement_missing_fields(self):
        """Test handling of participants with missing fields."""
        participants = [
            {"riotIdGameName": "Player1"},  # Missing tagline and placement
            {"riotIdTagline": "NA2", "placement": 2},  # Missing game name
            {"riotIdGameName": "Player3", "riotIdTagline": "NA3"},  # Missing placement
        ]
        
        match_info = TFTMatchInfo(
            match_id="NA1_123",
            game_creation=1234567890,
            game_datetime=1234567890,
            game_length=1800.0,
            game_variation=None,
            game_version="13.24",
            participants=participants,
            queue_id=1160,  # Double Up
            tft_game_type="standard",
            tft_set_number=10
        )
        
        # Should return None for all these cases
        assert match_info.get_placement_by_name("Player1", "NA1") is None
        assert match_info.get_placement_by_name("Player2", "NA2") is None
        assert match_info.get_placement_by_name("Player3", "NA3") is None