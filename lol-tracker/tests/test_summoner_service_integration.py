"""Integration tests for SummonerTrackingService."""

import pytest
from unittest.mock import patch, AsyncMock
from google.protobuf.timestamp_pb2 import Timestamp

from lol_tracker.summoner_service import SummonerTrackingService
from lol_tracker.proto.services import summoner_service_pb2
from lol_tracker.riot_api_client import (
    SummonerInfo,
    SummonerNotFoundError,
    InvalidRegionError,
    RateLimitError,
    RiotAPIError,
)
from lol_tracker.database.repository import TrackedPlayerRepository
from tests.factories import TrackedPlayerFactory
from tests.riot_api_mocks import RiotAPIMockData
from tests.grpc_fixtures import summoner_service  # Import the fixture


@pytest.mark.integration
class TestSummonerTrackingServiceIntegration:
    """Integration tests for SummonerTrackingService with real database."""

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_success(
        self, summoner_service: SummonerTrackingService
    ):
        """Test successful summoner tracking with new summoner."""
        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API response
        mock_summoner_info = SummonerInfo(
            puuid="test_puuid_123",
            game_name="TestSummoner",
            tag_line="gamba",
        )

        # Patch the Riot API client method
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            return_value=mock_summoner_info,
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called correctly
            mock_get_summoner.assert_called_once_with("TestSummoner", "gamba")

            # Verify response
            assert response.success is True
            assert response.summoner_details.game_name == "TestSummoner"
            assert response.summoner_details.puuid == "test_puuid_123"
            # Note: summoner_level is no longer available in simplified API
            assert not response.HasField("error_message")
            assert not response.HasField("error_code")

            # Verify summoner was stored in database
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                tracked_player = await repo.get_by_puuid(
                    "test_puuid_123"
                )
            assert tracked_player is not None
            assert tracked_player.is_active is True
            assert tracked_player.puuid == "test_puuid_123"

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_not_found(
        self, summoner_service: SummonerTrackingService
    ):
        """Test summoner tracking when summoner is not found."""
        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="NonExistentSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API to raise SummonerNotFoundError
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            side_effect=SummonerNotFoundError("Summoner not found"),
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called
            mock_get_summoner.assert_called_once_with("NonExistentSummoner", "gamba")

            # Verify response
            assert response.success is False
            assert "NonExistentSummoner" in response.error_message
            assert "not found" in response.error_message
            assert (
                response.error_code
                == summoner_service_pb2.ValidationError.VALIDATION_ERROR_SUMMONER_NOT_FOUND
            )
            assert not response.HasField("summoner_details")

            # Verify no summoner was stored in database
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                tracked_player = await repo.get_by_puuid(
                    "nonexistent_puuid"
                )
            assert tracked_player is None

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_invalid_region(
        self, summoner_service: SummonerTrackingService
    ):
        """Test summoner tracking with invalid region."""
        # Create request with invalid region
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="INVALID",
            requested_at=timestamp,
        )

        # Mock Riot API to raise InvalidRegionError
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            side_effect=InvalidRegionError("Invalid region: invalid_region"),
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called
            mock_get_summoner.assert_called_once_with("TestSummoner", "INVALID")

            # Verify response
            assert response.success is False
            assert "Invalid region" in response.error_message
            assert (
                response.error_code
                == summoner_service_pb2.ValidationError.VALIDATION_ERROR_INVALID_REGION
            )
            assert not response.HasField("summoner_details")

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_rate_limited(
        self, summoner_service: SummonerTrackingService
    ):
        """Test summoner tracking when rate limited."""
        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API to raise RateLimitError
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            side_effect=RateLimitError("Rate limited. Retry after 60 seconds"),
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called
            mock_get_summoner.assert_called_once_with("TestSummoner", "gamba")

            # Verify response
            assert response.success is False
            assert "Rate limited" in response.error_message
            assert (
                response.error_code
                == summoner_service_pb2.ValidationError.VALIDATION_ERROR_RATE_LIMITED
            )
            assert not response.HasField("summoner_details")

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_api_error(
        self, summoner_service: SummonerTrackingService
    ):
        """Test summoner tracking when Riot API returns error."""
        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API to raise RiotAPIError
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            side_effect=RiotAPIError("API error: 500"),
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called
            mock_get_summoner.assert_called_once_with("TestSummoner", "gamba")

            # Verify response
            assert response.success is False
            assert "Riot API error" in response.error_message
            assert (
                response.error_code
                == summoner_service_pb2.ValidationError.VALIDATION_ERROR_API_ERROR
            )
            assert not response.HasField("summoner_details")

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_already_tracked(
        self, summoner_service: SummonerTrackingService, clean_db_session
    ):
        """Test tracking a summoner that is already being tracked."""
        # Create an existing tracked player
        repo = TrackedPlayerRepository(clean_db_session)
        existing_player = await repo.create(
            game_name="TestSummoner",
            tag_line="gamba",
            puuid="test_puuid_123",
        )
        await clean_db_session.commit()

        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API response
        mock_summoner_info = SummonerInfo(
            puuid="test_puuid_123",
            game_name="TestSummoner",
            tag_line="gamba",
        )

        # Patch the Riot API client method
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            return_value=mock_summoner_info,
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called (validation still happens)
            mock_get_summoner.assert_called_once_with("TestSummoner", "gamba")

            # Verify response returns the summoner information as normal. Endpoint is idempotent.
            assert response.success is True
            assert response.HasField("summoner_details")

    @pytest.mark.asyncio
    async def test_start_tracking_summoner_reactivate_inactive(
        self, summoner_service: SummonerTrackingService, clean_db_session
    ):
        """Test reactivating an inactive tracked summoner."""
        # Create an inactive tracked player
        repo = TrackedPlayerRepository(clean_db_session)
        inactive_player = await repo.create(
            game_name="TestSummoner",
            tag_line="gamba",
            puuid="test_puuid_123",
        )

        # Deactivate the player
        await repo.set_active_status(inactive_player.id, False)
        await clean_db_session.commit()

        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API response
        mock_summoner_info = SummonerInfo(
            puuid="test_puuid_123",
            game_name="TestSummoner",
            tag_line="gamba",
        )

        # Patch the Riot API client method
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            return_value=mock_summoner_info,
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called
            mock_get_summoner.assert_called_once_with("TestSummoner", "gamba")

            # Verify response indicates success
            assert response.success is True
            assert response.summoner_details.game_name == "TestSummoner"
            assert response.summoner_details.puuid == "test_puuid_123"
            assert not response.HasField("error_message")

            # Verify player was reactivated and updated
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                updated_player = await repo.get_by_puuid(
                    "test_puuid_123"
                )
            assert updated_player is not None
            assert updated_player.is_active is True
            assert updated_player.puuid == "test_puuid_123"
            assert updated_player.id == inactive_player.id  # Same player, just updated

    @pytest.mark.asyncio
    async def test_stop_tracking_summoner_success(
        self, summoner_service: SummonerTrackingService, clean_db_session
    ):
        """Test successful summoner stop tracking."""
        # Create an active tracked player
        repo = TrackedPlayerRepository(clean_db_session)
        tracked_player = await repo.create(
            game_name="TestSummoner",
            tag_line="gamba",
            puuid="test_puuid",
        )
        await clean_db_session.commit()

        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StopTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API response for validation
        mock_summoner_info = SummonerInfo(
            puuid="test_puuid",
            game_name="TestSummoner",
            tag_line="gamba",
        )

        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            return_value=mock_summoner_info,
        ):
            # Call the service method
            response = await summoner_service.StopTrackingSummoner(request, None)

        # Verify response
        assert response.success is True
        assert not response.HasField("error_message")
        assert not response.HasField("error_code")

        # Verify player was deactivated
        async with summoner_service.db_manager.get_session() as session:
            repo = TrackedPlayerRepository(session)
            updated_player = await repo.get_by_puuid(
                "test_puuid"
            )
        assert updated_player is not None
        assert updated_player.is_active is False
        assert updated_player.id == tracked_player.id

    @pytest.mark.asyncio
    async def test_stop_tracking_summoner_not_tracked(
        self, summoner_service: SummonerTrackingService
    ):
        """Test stopping tracking for summoner that is not tracked."""
        # Create request for non-tracked summoner
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StopTrackingSummonerRequest(
            game_name="NonTrackedSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API response for validation
        mock_summoner_info = SummonerInfo(
            puuid="nontracked_puuid",
            game_name="NonTrackedSummoner",
            tag_line="gamba",
        )

        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            return_value=mock_summoner_info,
        ):
            # Call the service method
            response = await summoner_service.StopTrackingSummoner(request, None)

        # Verify response indicates not tracked
        assert response.success is False
        assert "not currently being tracked" in response.error_message
        assert (
            response.error_code
            == summoner_service_pb2.ValidationError.VALIDATION_ERROR_NOT_TRACKED
        )

    @pytest.mark.asyncio
    async def test_stop_tracking_summoner_already_inactive(
        self, summoner_service: SummonerTrackingService, clean_db_session
    ):
        """Test stopping tracking for summoner that is already inactive."""
        # Create an inactive tracked player
        repo = TrackedPlayerRepository(clean_db_session)
        tracked_player = await repo.create(
            game_name="TestSummoner",
            tag_line="gamba",
            puuid="test_puuid",
        )

        # Deactivate the player
        await repo.set_active_status(tracked_player.id, False)
        await clean_db_session.commit()

        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StopTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock Riot API response for validation
        mock_summoner_info = SummonerInfo(
            puuid="test_puuid",
            game_name="TestSummoner",
            tag_line="gamba",
        )

        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            return_value=mock_summoner_info,
        ):
            # Call the service method
            response = await summoner_service.StopTrackingSummoner(request, None)

        # Verify response indicates not tracked
        assert response.success is False
        assert "not currently being tracked" in response.error_message
        assert (
            response.error_code
            == summoner_service_pb2.ValidationError.VALIDATION_ERROR_NOT_TRACKED
        )

    @pytest.mark.asyncio
    async def test_internal_error_handling(
        self, summoner_service: SummonerTrackingService
    ):
        """Test internal error handling."""
        # Create request
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        request = summoner_service_pb2.StartTrackingSummonerRequest(
            game_name="TestSummoner", tag_line="gamba", requested_at=timestamp
        )

        # Mock an unexpected exception
        with patch.object(
            summoner_service.riot_api_client,
            "get_summoner_by_name",
            side_effect=Exception("Unexpected error"),
        ) as mock_get_summoner:

            # Call the service method
            response = await summoner_service.StartTrackingSummoner(request, None)

            # Verify API was called
            mock_get_summoner.assert_called_once_with("TestSummoner", "gamba")

            # Verify response indicates internal error
            assert response.success is False
            assert "Internal service error" in response.error_message
            assert (
                response.error_code
                == summoner_service_pb2.ValidationError.VALIDATION_ERROR_INTERNAL_ERROR
            )
            assert not response.HasField("summoner_details")
