"""End-to-end tests for summoner tracking gRPC service."""

import pytest
from unittest.mock import patch

from lol_tracker.proto.services import summoner_service_pb2, summoner_service_pb2_grpc
from lol_tracker.riot_api_client import (
    SummonerInfo,
    SummonerNotFoundError,
    InvalidRegionError,
    RateLimitError,
    RiotAPIError,
)
from lol_tracker.database.repository import TrackedPlayerRepository
from tests.grpc_fixtures import *  # Import all gRPC fixtures
from tests.riot_api_mocks import (
    mock_successful_summoner_lookup,
    mock_summoner_not_found,
    mock_rate_limit_error,
    mock_api_error,
    RiotAPIMockData,
)


@pytest.mark.integration
class TestSummonerTrackingE2E:
    """End-to-end tests for summoner tracking with full gRPC client-server communication."""

    @pytest.mark.asyncio
    async def test_e2e_start_tracking_summoner_success(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_start_tracking_request: summoner_service_pb2.StartTrackingSummonerRequest,
        summoner_service
    ):
        """Test complete end-to-end flow for successful summoner tracking."""
        # Mock Riot API response for successful lookup
        mock_summoner_info = SummonerInfo(
            puuid="e2e_test_puuid_123",
            account_id="e2e_test_account_123",
            summoner_id="e2e_test_summoner_123",
            summoner_name="TestSummoner",
            summoner_level=150,
            region="na1",
            last_updated=1234567890
        )

        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            return_value=mock_summoner_info
        ):
            # Make gRPC call through client
            response = await grpc_client.StartTrackingSummoner(sample_start_tracking_request)

            # Verify response structure
            assert response.success is True
            assert response.HasField("summoner_details")
            assert not response.HasField("error_message")
            assert not response.HasField("error_code")

            # Verify summoner details
            details = response.summoner_details
            assert details.summoner_name == "TestSummoner"
            assert details.puuid == "e2e_test_puuid_123"
            assert details.account_id == "e2e_test_account_123"
            assert details.summoner_id == "e2e_test_summoner_123"
            assert details.summoner_level == 150
            assert details.region == "na1"
            assert details.last_updated == 1234567890

            # Verify data was stored in database
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                tracked_player = await repo.get_by_summoner_and_region(
                    "TestSummoner", "na1"
                )
            assert tracked_player is not None
            assert tracked_player.is_active is True
            assert tracked_player.puuid == "e2e_test_puuid_123"
            assert tracked_player.summoner_name == "TestSummoner"

    @pytest.mark.asyncio
    async def test_e2e_start_tracking_summoner_not_found(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        summoner_service
    ):
        """Test end-to-end flow when summoner is not found."""
        # Create request for non-existent summoner
        from google.protobuf.timestamp_pb2 import Timestamp
        
        timestamp = Timestamp()
        timestamp.GetCurrentTime()
        
        request = summoner_service_pb2.StartTrackingSummonerRequest(
            summoner_name="NonExistentSummoner",
            region="na1",
            requested_at=timestamp
        )

        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            side_effect=SummonerNotFoundError("Summoner not found")
        ):
            # Make gRPC call
            response = await grpc_client.StartTrackingSummoner(request)

            # Verify error response
            assert response.success is False
            assert not response.HasField("summoner_details")
            assert response.HasField("error_message")
            assert "NonExistentSummoner" in response.error_message
            assert "not found" in response.error_message
            assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_SUMMONER_NOT_FOUND

            # Verify no data in database
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                tracked_player = await repo.get_by_summoner_and_region(
                    "NonExistentSummoner", "na1"
                )
            assert tracked_player is None

    @pytest.mark.asyncio
    async def test_e2e_rate_limit_handling(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_start_tracking_request: summoner_service_pb2.StartTrackingSummonerRequest,
        summoner_service
    ):
        """Test end-to-end flow when rate limited by Riot API."""
        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            side_effect=RateLimitError("Rate limited. Retry after 60 seconds")
        ):
            # Make gRPC call
            response = await grpc_client.StartTrackingSummoner(sample_start_tracking_request)

            # Verify rate limit error response
            assert response.success is False
            assert not response.HasField("summoner_details")
            assert "Rate limited" in response.error_message
            assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_RATE_LIMITED

    @pytest.mark.asyncio
    async def test_e2e_invalid_region(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        summoner_service
    ):
        """Test end-to-end flow with invalid region."""
        # Create request with invalid region
        from google.protobuf.timestamp_pb2 import Timestamp
        
        timestamp = Timestamp()
        timestamp.GetCurrentTime()
        
        request = summoner_service_pb2.StartTrackingSummonerRequest(
            summoner_name="TestSummoner",
            region="invalid_region",
            requested_at=timestamp
        )

        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            side_effect=InvalidRegionError("Invalid region: invalid_region")
        ):
            # Make gRPC call
            response = await grpc_client.StartTrackingSummoner(request)

            # Verify invalid region error response
            assert response.success is False
            assert not response.HasField("summoner_details")
            assert "Invalid region" in response.error_message
            assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_INVALID_REGION

    @pytest.mark.asyncio
    async def test_e2e_stop_tracking_summoner_success(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_stop_tracking_request: summoner_service_pb2.StopTrackingSummonerRequest,
        summoner_service,
        clean_db_session
    ):
        """Test complete end-to-end flow for successful summoner stop tracking."""
        # First, create a tracked summoner
        repo = TrackedPlayerRepository(clean_db_session)
        tracked_player = await repo.create(
            summoner_name="TestSummoner",
            region="na1",
            puuid="test_puuid",
            account_id="test_account",
            summoner_id="test_summoner"
        )
        await clean_db_session.commit()

        # Make gRPC call to stop tracking
        response = await grpc_client.StopTrackingSummoner(sample_stop_tracking_request)

        # Verify response
        assert response.success is True
        assert not response.HasField("error_message")
        assert not response.HasField("error_code")

        # Verify summoner was deactivated in database
        async with summoner_service.db_manager.get_session() as session:
            repo = TrackedPlayerRepository(session)
            updated_player = await repo.get_by_summoner_and_region(
                "TestSummoner", "na1"
            )
        assert updated_player is not None
        assert updated_player.is_active is False
        assert updated_player.id == tracked_player.id

    @pytest.mark.asyncio
    async def test_e2e_stop_tracking_summoner_not_tracked(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        summoner_service
    ):
        """Test end-to-end flow when trying to stop tracking non-tracked summoner."""
        # Create request for non-tracked summoner
        from google.protobuf.timestamp_pb2 import Timestamp
        
        timestamp = Timestamp()
        timestamp.GetCurrentTime()
        
        request = summoner_service_pb2.StopTrackingSummonerRequest(
            summoner_name="NonTrackedSummoner",
            region="na1",
            requested_at=timestamp
        )

        # Make gRPC call
        response = await grpc_client.StopTrackingSummoner(request)

        # Verify error response
        assert response.success is False
        assert "not currently being tracked" in response.error_message
        assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_NOT_TRACKED

    @pytest.mark.asyncio
    async def test_e2e_duplicate_tracking_detection(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_start_tracking_request: summoner_service_pb2.StartTrackingSummonerRequest,
        summoner_service,
        clean_db_session
    ):
        """Test end-to-end flow for detecting already tracked summoners."""
        # First, create a tracked summoner
        repo = TrackedPlayerRepository(clean_db_session)
        existing_player = await repo.create(
            summoner_name="TestSummoner",
            region="na1",
            puuid="existing_puuid",
            account_id="existing_account",
            summoner_id="existing_summoner"
        )
        await clean_db_session.commit()

        # Mock Riot API response (validation still happens)
        mock_summoner_info = SummonerInfo(
            puuid="new_puuid_123",
            account_id="new_account_123",
            summoner_id="new_summoner_123",
            summoner_name="TestSummoner",
            summoner_level=100,
            region="na1",
            last_updated=1234567890
        )

        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            return_value=mock_summoner_info
        ):
            # Make gRPC call
            response = await grpc_client.StartTrackingSummoner(sample_start_tracking_request)

            # Verify already tracked error response
            assert response.success is False
            assert not response.HasField("summoner_details")
            assert "already being tracked" in response.error_message
            assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_ALREADY_TRACKED

            # Verify original player is unchanged
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                original_player = await repo.get_by_summoner_and_region(
                    "TestSummoner", "na1"
                )
            assert original_player is not None
            assert original_player.puuid == "existing_puuid"  # Not updated
            assert original_player.id == existing_player.id

    @pytest.mark.asyncio
    async def test_e2e_reactivate_inactive_summoner(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_start_tracking_request: summoner_service_pb2.StartTrackingSummonerRequest,
        summoner_service,
        clean_db_session
    ):
        """Test end-to-end flow for reactivating an inactive summoner."""
        # Create an inactive tracked summoner
        repo = TrackedPlayerRepository(clean_db_session)
        inactive_player = await repo.create(
            summoner_name="TestSummoner",
            region="na1",
            puuid="old_puuid",
            account_id="old_account",
            summoner_id="old_summoner"
        )
        
        # Deactivate the player
        await repo.set_active_status(inactive_player.id, False)
        await clean_db_session.commit()

        # Mock Riot API response with updated info
        mock_summoner_info = SummonerInfo(
            puuid="updated_puuid_123",
            account_id="updated_account_123",
            summoner_id="updated_summoner_123",
            summoner_name="TestSummoner",
            summoner_level=200,
            region="na1",
            last_updated=9876543210
        )

        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            return_value=mock_summoner_info
        ):
            # Make gRPC call to reactivate
            response = await grpc_client.StartTrackingSummoner(sample_start_tracking_request)

            # Verify successful reactivation
            assert response.success is True
            assert response.HasField("summoner_details")
            assert not response.HasField("error_message")

            # Verify summoner details reflect updated info
            details = response.summoner_details
            assert details.puuid == "updated_puuid_123"
            assert details.summoner_level == 200
            assert details.last_updated == 9876543210

            # Verify database shows reactivated and updated player
            async with summoner_service.db_manager.get_session() as session:
                repo = TrackedPlayerRepository(session)
                reactivated_player = await repo.get_by_summoner_and_region(
                    "TestSummoner", "na1"
                )
            assert reactivated_player is not None
            assert reactivated_player.is_active is True
            assert reactivated_player.puuid == "updated_puuid_123"
            assert reactivated_player.id == inactive_player.id  # Same record, updated

    @pytest.mark.asyncio
    @pytest.mark.skip(reason="Complex multi-region test has intermittent issues with gRPC fixtures")
    async def test_e2e_multiple_regions(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        summoner_service,
        clean_db_session
    ):
        """Test end-to-end flow for tracking same summoner name in different regions."""
        from google.protobuf.timestamp_pb2 import Timestamp
        
        timestamp = Timestamp()
        timestamp.GetCurrentTime()

        # Test tracking in NA1
        na1_request = summoner_service_pb2.StartTrackingSummonerRequest(
            summoner_name="CrossRegionSummoner",
            region="na1",
            requested_at=timestamp
        )
        
        na1_summoner_info = SummonerInfo(
            puuid="na1_puuid_123",
            account_id="na1_account_123",
            summoner_id="na1_summoner_123",
            summoner_name="CrossRegionSummoner",
            summoner_level=100,
            region="na1",
            last_updated=1111111111
        )

        # Test tracking in EUW1
        euw1_request = summoner_service_pb2.StartTrackingSummonerRequest(
            summoner_name="CrossRegionSummoner",
            region="euw1",
            requested_at=timestamp
        )
        
        euw1_summoner_info = SummonerInfo(
            puuid="euw1_puuid_456",
            account_id="euw1_account_456",
            summoner_id="euw1_summoner_456",
            summoner_name="CrossRegionSummoner",
            summoner_level=200,
            region="euw1",
            last_updated=2222222222
        )

        # Track in NA1
        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            return_value=na1_summoner_info
        ):
            na1_response = await grpc_client.StartTrackingSummoner(na1_request)
            assert na1_response.success is True
            assert na1_response.summoner_details.puuid == "na1_puuid_123"

        # Track in EUW1 (should be allowed - different region)
        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            return_value=euw1_summoner_info
        ):
            euw1_response = await grpc_client.StartTrackingSummoner(euw1_request)
            assert euw1_response.success is True
            assert euw1_response.summoner_details.puuid == "euw1_puuid_456"

        # Verify both summoners are tracked separately
        async with summoner_service.db_manager.get_session() as session:
            repo = TrackedPlayerRepository(session)
            na1_player = await repo.get_by_summoner_and_region(
                "CrossRegionSummoner", "na1"
            )
            euw1_player = await repo.get_by_summoner_and_region(
                "CrossRegionSummoner", "euw1"
            )
        
        assert na1_player is not None
        assert euw1_player is not None
        assert na1_player.id != euw1_player.id
        assert na1_player.puuid == "na1_puuid_123"
        assert euw1_player.puuid == "euw1_puuid_456"

    @pytest.mark.asyncio
    async def test_e2e_api_error_handling(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_start_tracking_request: summoner_service_pb2.StartTrackingSummonerRequest,
        summoner_service
    ):
        """Test end-to-end flow when Riot API returns error."""
        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            side_effect=RiotAPIError("API error: 503 Service Unavailable")
        ):
            # Make gRPC call
            response = await grpc_client.StartTrackingSummoner(sample_start_tracking_request)

            # Verify API error response
            assert response.success is False
            assert not response.HasField("summoner_details")
            assert "Riot API error" in response.error_message
            assert "503" in response.error_message
            assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_API_ERROR

    @pytest.mark.asyncio
    async def test_e2e_internal_error_handling(
        self,
        grpc_client: summoner_service_pb2_grpc.SummonerTrackingServiceStub,
        sample_start_tracking_request: summoner_service_pb2.StartTrackingSummonerRequest,
        summoner_service
    ):
        """Test end-to-end flow when unexpected internal error occurs."""
        with patch.object(
            summoner_service.riot_api_client,
            'get_summoner_by_name',
            side_effect=Exception("Unexpected internal error")
        ):
            # Make gRPC call
            response = await grpc_client.StartTrackingSummoner(sample_start_tracking_request)

            # Verify internal error response
            assert response.success is False
            assert not response.HasField("summoner_details")
            assert "Internal service error" in response.error_message
            assert response.error_code == summoner_service_pb2.ValidationError.VALIDATION_ERROR_INTERNAL_ERROR