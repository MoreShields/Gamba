"""Focused tests for game detection queue filtering."""

import logging
from datetime import datetime
from types import SimpleNamespace
from unittest.mock import AsyncMock

import pytest

from lol_tracker.application import game_centric_polling_service
from lol_tracker.core.entities import Player
from lol_tracker.core.enums import QueueType


def _build_service(queue_id: int):
    game_data = {
        "gameId": 1234567890,
        "gameQueueConfigId": queue_id,
        "game_type": "LOL",
    }
    database = SimpleNamespace(
        get_tracked_game=AsyncMock(return_value=None),
        create_tracked_game=AsyncMock(
            return_value=SimpleNamespace(
                id=11,
                detected_at=datetime(2026, 1, 2, 3, 4, 5),
                started_at=datetime(2026, 1, 2, 3, 4, 5),
            )
        ),
    )
    riot_api = SimpleNamespace(
        get_active_game_info=AsyncMock(
            return_value=SimpleNamespace(to_dict=lambda: game_data)
        )
    )
    event_publisher = SimpleNamespace(publish_game_state_changed=AsyncMock())
    config = SimpleNamespace(
        detection_interval_seconds=30,
        completion_interval_seconds=60,
    )
    service = game_centric_polling_service.GameCentricPollingService(
        database=database,
        riot_api=riot_api,
        event_publisher=event_publisher,
        config=config,
    )
    return service, database, event_publisher


def test_supported_queue_types_match_discord_handlers():
    supported = {queue for queue in QueueType if queue.is_supported}

    assert supported == {
        QueueType.RANKED_SOLO_5X5,
        QueueType.RANKED_FLEX_SR,
        QueueType.TFT_RANKED,
        QueueType.TFT_RANKED_DOUBLE_UP,
    }


@pytest.mark.asyncio
@pytest.mark.parametrize("queue_id", [400, 2400])
async def test_unsupported_game_is_ignored_before_database_lookup_or_event(
    queue_id, caplog
):
    service, database, event_publisher = _build_service(queue_id)
    player = Player(game_name="UnsupportedPlayer", tag_line="NA1", id=7)

    with caplog.at_level(logging.INFO):
        detected = await service._detect_game_for_player(player)

    assert detected is False
    database.get_tracked_game.assert_not_awaited()
    database.create_tracked_game.assert_not_awaited()
    event_publisher.publish_game_state_changed.assert_not_awaited()
    assert (
        f"Ignoring unsupported queue {queue_id} for game 1234567890 "
        "and player UnsupportedPlayer#NA1"
    ) in caplog.messages


@pytest.mark.asyncio
async def test_supported_game_is_tracked_and_published():
    service, database, event_publisher = _build_service(440)
    player = Player(game_name="SupportedPlayer", tag_line="NA1", id=8)

    detected = await service._detect_game_for_player(player)

    assert detected is True
    database.get_tracked_game.assert_awaited_once_with(8, "1234567890")
    database.create_tracked_game.assert_awaited_once()
    created_game = database.create_tracked_game.await_args
    assert created_game.kwargs["queue_type"] == "RANKED_FLEX_SR"
    event_publisher.publish_game_state_changed.assert_awaited_once()
