"""Simplified NATS event publishing infrastructure layer."""

import json
import logging
from typing import Optional, Dict, Any
from datetime import datetime

import nats
from nats.aio.client import Client as NATSClient
from nats.js import JetStreamContext
import nats.js.errors
from google.protobuf.timestamp_pb2 import Timestamp

from ...config import Config
from ...proto.events import lol_events_pb2

logger = logging.getLogger(__name__)


class EventPublisher:
    """Simple NATS event publisher for LoL Tracker events."""

    def __init__(self, config: Config):
        """Initialize the event publisher.
        
        Args:
            config: Application configuration
        """
        self.config = config
        self._client: Optional[NATSClient] = None
        self._js: Optional[JetStreamContext] = None
        self._connected = False

    async def initialize(self) -> None:
        """Initialize connection to NATS."""
        if self._connected:
            logger.warning("Event publisher already initialized")
            return

        try:
            logger.info(f"Connecting to NATS at {self.config.message_bus_url}")

            self._client = await nats.connect(
                servers=self.config.message_bus_url,
                connect_timeout=self.config.message_bus_timeout_seconds,
                max_reconnect_attempts=self.config.message_bus_max_reconnect_attempts,
                reconnect_time_wait=self.config.message_bus_reconnect_delay_seconds,
                error_cb=self._error_callback,
                disconnected_cb=self._disconnected_callback,
                reconnected_cb=self._reconnected_callback,
            )

            # Enable JetStream
            self._js = self._client.jetstream()
            self._connected = True

            # Create streams
            await self._create_streams()

            logger.info("Event publisher initialized successfully")

        except Exception as e:
            logger.error(f"Failed to initialize event publisher: {e}")
            self._connected = False
            raise

    async def close(self) -> None:
        """Close connection to NATS."""
        if not self._connected or not self._client:
            return

        try:
            logger.info("Closing event publisher")
            await self._client.close()
            self._connected = False
            self._client = None
            self._js = None
            logger.info("Event publisher closed")
        except Exception as e:
            logger.error(f"Error closing event publisher: {e}")

    async def _create_streams(self) -> None:
        """Create required JetStream streams if they don't exist."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        streams_config = [
            {
                "name": self.config.lol_events_stream,
                "subjects": [f"{self.config.game_state_events_subject}.state_changed"],
                "description": "LoL game state change events (consolidated)",
                "retention": "limits",
                "max_age": self.config.jetstream_max_age_hours * 60 * 60,  # Convert to seconds
                "max_msgs": self.config.jetstream_max_msgs_lol,
                "storage": self.config.jetstream_storage,
            },
            {
                "name": self.config.tracking_events_stream,
                "subjects": [f"{self.config.tracking_events_subject}.*"],
                "description": "Player tracking command events",
                "retention": "limits",
                "max_age": self.config.jetstream_max_age_hours * 60 * 60,  # Convert to seconds
                "max_msgs": self.config.jetstream_max_msgs_tracking,
                "storage": self.config.jetstream_storage,
            },
        ]

        for stream_config in streams_config:
            stream_name = stream_config["name"]
            try:
                # Check if stream already exists
                await self._js.stream_info(stream_name)
                logger.info(f"JetStream stream '{stream_name}' already exists")
            except nats.js.errors.NotFoundError:
                # Stream doesn't exist, create it
                logger.info(f"Creating JetStream stream '{stream_name}'")
                await self._js.add_stream(
                    name=stream_name,
                    subjects=stream_config["subjects"],
                    description=stream_config["description"],
                    retention=stream_config["retention"],
                    max_age=stream_config["max_age"],
                    max_msgs=stream_config["max_msgs"],
                    storage=stream_config["storage"],
                )
                logger.info(f"Successfully created JetStream stream '{stream_name}'")
            except Exception as e:
                logger.error(f"Failed to create/verify stream '{stream_name}': {e}")
                raise

    async def is_healthy(self) -> bool:
        """Check if the event publisher is healthy and can publish events."""
        return (
            self._connected and self._client is not None and self._client.is_connected
        )

    # Direct event publishing methods for common events
    
    async def publish_player_tracking_started(
        self,
        player_id: int,
        game_name: str,
        tag_line: str,
        puuid: str,
        started_at: datetime,
    ) -> None:
        """Publish player tracking started event."""
        subject = f"{self.config.tracking_events_subject}.started"
        message_data = {
            "event_type": "PlayerTrackingStarted",
            "player_id": player_id,
            "summoner_identity": {
                "game_name": game_name,
                "tag_line": tag_line,
                "puuid": puuid,
            },
            "started_at": started_at.isoformat(),
            "timestamp": datetime.utcnow().isoformat(),
        }
        
        await self._publish_json_message(subject, message_data)

    async def publish_player_tracking_stopped(
        self,
        player_id: int,
        game_name: str,
        tag_line: str,
        puuid: str,
        stopped_at: datetime,
        reason: str = "manual",
    ) -> None:
        """Publish player tracking stopped event."""
        subject = f"{self.config.tracking_events_subject}.stopped"
        message_data = {
            "event_type": "PlayerTrackingStopped",
            "player_id": player_id,
            "summoner_identity": {
                "game_name": game_name,
                "tag_line": tag_line,
                "puuid": puuid,
            },
            "stopped_at": stopped_at.isoformat(),
            "reason": reason,
            "timestamp": datetime.utcnow().isoformat(),
        }
        
        await self._publish_json_message(subject, message_data)

    async def publish_game_state_changed(
        self,
        player_id: int,
        game_name: str,
        tag_line: str,
        puuid: str,
        previous_status: str,
        new_status: str,
        game_id: Optional[str] = None,
        queue_type: Optional[str] = None,
        changed_at: Optional[datetime] = None,
        is_game_start: bool = False,
        is_game_end: bool = False,
        # Game result fields (only set when game ends with results)
        won: Optional[bool] = None,
        duration_seconds: Optional[int] = None,
        champion_played: Optional[str] = None,
    ) -> None:
        """Publish game state changed event as protobuf with optional game results."""
        # Create protobuf message
        event = lol_events_pb2.LoLGameStateChanged()
        event.game_name = game_name
        event.tag_line = tag_line
        
        # Map status strings to protobuf enum values
        if previous_status == "NOT_IN_GAME":
            event.previous_status = lol_events_pb2.GAME_STATUS_NOT_IN_GAME
        elif previous_status == "IN_GAME":
            event.previous_status = lol_events_pb2.GAME_STATUS_IN_GAME
            
        if new_status == "NOT_IN_GAME":
            event.current_status = lol_events_pb2.GAME_STATUS_NOT_IN_GAME
        elif new_status == "IN_GAME":
            event.current_status = lol_events_pb2.GAME_STATUS_IN_GAME
            
        # Set optional fields
        if game_id:
            event.game_id = game_id
        if queue_type:
            event.queue_type = queue_type
            
        # Set timestamp
        timestamp = Timestamp()
        timestamp.FromDatetime(changed_at or datetime.utcnow())
        event.event_time.CopyFrom(timestamp)
        
        # Set game result if provided (when game ends with complete results)
        if is_game_end and won is not None and duration_seconds is not None and champion_played is not None:
            game_result = lol_events_pb2.GameResult()
            game_result.won = won
            game_result.duration_seconds = duration_seconds
            game_result.champion_played = champion_played
            if queue_type:
                game_result.queue_type = queue_type
            event.game_result.CopyFrom(game_result)
        
        # Always publish to state_changed subject - no separate completed subject
        subject = f"{self.config.game_state_events_subject}.state_changed"
        await self._publish_protobuf_message(subject, event)


    # Generic message publishing
    
    async def publish_message(self, subject: str, data: Dict[str, Any]) -> None:
        """Publish a generic message with JSON data."""
        await self._publish_json_message(subject, data)

    async def _publish_json_message(self, subject: str, data: Dict[str, Any]) -> None:
        """Publish a JSON message to a NATS subject."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        try:
            json_data = json.dumps(data, ensure_ascii=False).encode('utf-8')
            ack = await self._js.publish(subject, json_data)
            logger.debug(f"Published message to {subject}, ack: {ack}")
        except Exception as e:
            logger.error(f"Failed to publish message to {subject}: {e}")
            raise
    
    async def _publish_protobuf_message(self, subject: str, message) -> None:
        """Publish a protobuf message to a NATS subject."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        try:
            # Serialize protobuf to bytes
            message_bytes = message.SerializeToString()
            ack = await self._js.publish(subject, message_bytes)
            logger.debug(f"Published protobuf message to {subject}, ack: {ack}, type: {type(message).__name__}")
        except Exception as e:
            logger.error(f"Failed to publish protobuf message to {subject}: {e}")
            raise

    # NATS callbacks
    
    async def _error_callback(self, error):
        """Handle NATS connection errors."""
        logger.error(f"NATS error: {error}")

    async def _disconnected_callback(self):
        """Handle NATS disconnection."""
        logger.warning("Disconnected from NATS")
        self._connected = False

    async def _reconnected_callback(self):
        """Handle NATS reconnection."""
        logger.info("Reconnected to NATS")
        self._connected = True


class MockEventPublisher:
    """Mock event publisher for testing."""

    def __init__(self, config: Config):
        self.config = config
        self._connected = False
        self.published_messages = []

    async def initialize(self) -> None:
        """Mock initialize operation."""
        self._connected = True
        logger.info("Mock: Event publisher initialized")

    async def close(self) -> None:
        """Mock close operation."""
        self._connected = False
        logger.info("Mock: Event publisher closed")

    async def is_healthy(self) -> bool:
        """Mock health check."""
        return self._connected

    async def publish_player_tracking_started(
        self,
        player_id: int,
        game_name: str,
        tag_line: str,
        puuid: str,
        started_at: datetime,
    ) -> None:
        """Mock publish player tracking started event."""
        message = {
            "subject": f"{self.config.tracking_events_subject}.started",
            "event_type": "PlayerTrackingStarted",
            "player_id": player_id,
            "game_name": game_name,
            "tag_line": tag_line,
            "puuid": puuid,
            "started_at": started_at,
        }
        self.published_messages.append(message)
        logger.debug(f"Mock: Published player tracking started event for player {player_id}")

    async def publish_player_tracking_stopped(
        self,
        player_id: int,
        game_name: str,
        tag_line: str,
        puuid: str,
        stopped_at: datetime,
        reason: str = "manual",
    ) -> None:
        """Mock publish player tracking stopped event."""
        message = {
            "subject": f"{self.config.tracking_events_subject}.stopped",
            "event_type": "PlayerTrackingStopped",
            "player_id": player_id,
            "game_name": game_name,
            "tag_line": tag_line,
            "puuid": puuid,
            "stopped_at": stopped_at,
            "reason": reason,
        }
        self.published_messages.append(message)
        logger.debug(f"Mock: Published player tracking stopped event for player {player_id}")

    async def publish_game_state_changed(
        self,
        player_id: int,
        game_name: str,
        tag_line: str,
        puuid: str,
        previous_status: str,
        new_status: str,
        game_id: Optional[str] = None,
        queue_type: Optional[str] = None,
        changed_at: Optional[datetime] = None,
        is_game_start: bool = False,
        is_game_end: bool = False,
        won: Optional[bool] = None,
        duration_seconds: Optional[int] = None,
        champion_played: Optional[str] = None,
    ) -> None:
        """Mock publish game state changed event with optional game results."""
        message = {
            "subject": f"{self.config.game_state_events_subject}.state_changed",
            "event_type": "PlayerGameStateChanged",
            "player_id": player_id,
            "game_name": game_name,
            "tag_line": tag_line,
            "puuid": puuid,
            "previous_status": previous_status,
            "new_status": new_status,
            "game_id": game_id,
            "queue_type": queue_type,
            "changed_at": changed_at or datetime.utcnow(),
            "is_game_start": is_game_start,
            "is_game_end": is_game_end,
        }
        
        # Add game result if provided
        if is_game_end and won is not None and duration_seconds is not None and champion_played is not None:
            message["game_result"] = {
                "won": won,
                "duration_seconds": duration_seconds,
                "champion_played": champion_played,
                "queue_type": queue_type
            }
        
        self.published_messages.append(message)
        logger.debug(f"Mock: Published game state changed event for player {player_id}")


    async def publish_message(self, subject: str, data: Dict[str, Any]) -> None:
        """Mock publish generic message."""
        message = {"subject": subject, "data": data}
        self.published_messages.append(message)
        logger.debug(f"Mock: Published message to {subject}")


# Global event publisher instance
_event_publisher: Optional[EventPublisher] = None


def get_event_publisher() -> EventPublisher:
    """Get the global event publisher instance."""
    global _event_publisher
    if _event_publisher is None:
        raise RuntimeError(
            "Event publisher not initialized. Call initialize_event_publisher() first."
        )
    return _event_publisher


async def initialize_event_publisher(config: Config, use_mock: bool = False) -> EventPublisher:
    """Initialize the global event publisher."""
    global _event_publisher
    if _event_publisher is not None:
        logger.warning("Event publisher already initialized")
        return _event_publisher

    if use_mock or config.is_test():
        _event_publisher = MockEventPublisher(config)
    else:
        _event_publisher = EventPublisher(config)
    
    await _event_publisher.initialize()
    return _event_publisher


async def close_event_publisher() -> None:
    """Close the global event publisher."""
    global _event_publisher
    if _event_publisher is not None:
        await _event_publisher.close()
        _event_publisher = None