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
from ...proto.events import lol_events_pb2, tft_events_pb2
from ...core.events import GameStateChangedEvent, LoLGameStateChangedEvent, TFTGameStateChangedEvent
from ...core.enums import GameStatus

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
                "name": self.config.tft_events_stream,
                "subjects": [f"{self.config.tft_game_state_events_subject}.state_changed"],
                "description": "TFT game state change events",
                "retention": "limits",
                "max_age": self.config.jetstream_max_age_hours * 60 * 60,  # Convert to seconds
                "max_msgs": self.config.jetstream_max_msgs_lol,  # Reuse LoL limit for now
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
    
    # Tracking events are currently unused but kept for potential future use
    # If needed, these should be converted to domain events like game state changes

    # Queue type mapping for TFT - maps our internal queue types to protobuf enums
    def _map_game_status_to_lol_enum(self, status: str):
        """Map game status string to LoL protobuf enum."""
        if status == GameStatus.NOT_IN_GAME.value:
            return lol_events_pb2.GAME_STATUS_NOT_IN_GAME
        elif status == GameStatus.IN_GAME.value:
            return lol_events_pb2.GAME_STATUS_IN_GAME
        return lol_events_pb2.GAME_STATUS_NOT_IN_GAME
    
    def _map_game_status_to_tft_enum(self, status: str):
        """Map game status string to TFT protobuf enum."""
        if status == GameStatus.NOT_IN_GAME.value:
            return tft_events_pb2.TFT_GAME_STATUS_NOT_IN_GAME
        elif status == GameStatus.IN_GAME.value:
            return tft_events_pb2.TFT_GAME_STATUS_IN_GAME
        return tft_events_pb2.TFT_GAME_STATUS_NOT_IN_GAME

    def _set_common_protobuf_fields(self, pb_event, event: GameStateChangedEvent):
        """Set common fields for any protobuf game state event."""
        pb_event.game_name = event.game_name
        pb_event.tag_line = event.tag_line
        
        if event.game_id:
            pb_event.game_id = event.game_id
        if event.queue_type:
            pb_event.queue_type = event.queue_type
            
        # Set timestamp
        timestamp = Timestamp()
        timestamp.FromDatetime(event.changed_at)
        pb_event.event_time.CopyFrom(timestamp)

    async def publish_game_state_changed(self, event: GameStateChangedEvent) -> None:
        """Publish game state changed event as protobuf.
        
        Accepts polymorphic domain events and publishes them appropriately.
        
        Args:
            event: Domain event to publish (LoL or TFT specific)
        """
        if not self._connected:
            logger.warning("Cannot publish event - not connected to NATS")
            return
        
        # Route to appropriate handler based on event type
        if isinstance(event, TFTGameStateChangedEvent):
            await self._publish_tft_game_state_changed(event)
        else:
            # Default to LoL for LoLGameStateChangedEvent or unknown types
            await self._publish_lol_game_state_changed(event)
    
    async def _publish_lol_game_state_changed(self, event: GameStateChangedEvent) -> None:
        """Publish LoL game state changed event."""
        pb_event = lol_events_pb2.LoLGameStateChanged()
        
        # Set common fields
        self._set_common_protobuf_fields(pb_event, event)
        
        # Map status enums
        pb_event.previous_status = self._map_game_status_to_lol_enum(event.previous_status)
        pb_event.current_status = self._map_game_status_to_lol_enum(event.new_status)
        
        # Set game result if provided (when game ends with complete results)
        if event.is_game_end and event.duration_seconds is not None:
            if isinstance(event, LoLGameStateChangedEvent):
                if event.won is not None and event.champion_played is not None:
                    game_result = lol_events_pb2.GameResult()
                    game_result.won = event.won
                    game_result.duration_seconds = event.duration_seconds
                    game_result.champion_played = event.champion_played
                    if event.queue_type:
                        game_result.queue_type = event.queue_type
                    pb_event.game_result.CopyFrom(game_result)
        
        # Log the event details
        logger.info(
            f"Publishing LoL game state event - Player: {event.game_name}#{event.tag_line}, "
            f"Transition: {event.previous_status} -> {event.new_status}, "
            f"Game ID: {event.game_id or 'N/A'}, "
            f"Is game end: {event.is_game_end}"
        )
        
        # Publish to LoL subject
        subject = f"{self.config.game_state_events_subject}.state_changed"
        await self._publish_protobuf_message(subject, pb_event)
    
    async def _publish_tft_game_state_changed(self, event: TFTGameStateChangedEvent) -> None:
        """Publish TFT game state changed event."""
        pb_event = tft_events_pb2.TFTGameStateChanged()
        
        # Set common fields
        self._set_common_protobuf_fields(pb_event, event)
        
        # Map status enums
        pb_event.previous_status = self._map_game_status_to_tft_enum(event.previous_status)
        pb_event.current_status = self._map_game_status_to_tft_enum(event.new_status)
        
        # Set game result if provided
        if event.is_game_end and event.duration_seconds is not None and event.placement is not None:
            game_result = tft_events_pb2.TFTGameResult()
            game_result.placement = event.placement
            game_result.duration_seconds = event.duration_seconds
            pb_event.game_result.CopyFrom(game_result)
        
        # Log the event details
        logger.info(
            f"Publishing TFT game state event - Player: {event.game_name}#{event.tag_line}, "
            f"Transition: {event.previous_status} -> {event.new_status}, "
            f"Game ID: {event.game_id or 'N/A'}, "
            f"Is game end: {event.is_game_end}, "
            f"Placement: {event.placement if event.placement else 'N/A'}"
        )
        
        # Publish to TFT subject
        subject = f"{self.config.tft_game_state_events_subject}.state_changed"
        await self._publish_protobuf_message(subject, pb_event)
    async def _publish_protobuf_message(self, subject: str, message) -> None:
        """Publish a protobuf message to a NATS subject."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        try:
            # Serialize protobuf to bytes
            message_bytes = message.SerializeToString()
            ack = await self._js.publish(subject, message_bytes)
            logger.info(
                f"Successfully published to NATS - Subject: {subject}, "
                f"Message type: {type(message).__name__}, "
                f"Size: {len(message_bytes)} bytes, "
                f"Stream: {ack.stream}, Seq: {ack.seq}"
            )
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


class MockEventPublisher(EventPublisher):
    """Mock event publisher for testing that reuses real protobuf serialization logic."""

    def __init__(self, config: Config):
        super().__init__(config)
        self.published_messages = []  # Store (subject, protobuf_bytes, protobuf_message) tuples

    async def initialize(self) -> None:
        """Mock initialize operation - skip NATS connection."""
        self._connected = True
        logger.info("Mock: Event publisher initialized")

    async def close(self) -> None:
        """Mock close operation - no NATS to close."""
        self._connected = False
        logger.info("Mock: Event publisher closed")

    async def _create_streams(self) -> None:
        """Mock stream creation - no actual NATS streams."""
        logger.info("Mock: Skipping stream creation")

    async def _publish_protobuf_message(self, subject: str, message) -> None:
        """Override to capture protobuf messages instead of publishing to NATS."""
        try:
            # Serialize protobuf to bytes (validates serialization works)
            message_bytes = message.SerializeToString()
            
            # Store both the raw bytes and the message object for testing
            self.published_messages.append({
                "subject": subject,
                "protobuf_bytes": message_bytes,
                "protobuf_message": message,  # Keep the actual protobuf object for easy testing
                "message_type": type(message).__name__
            })
            
            logger.debug(f"Mock: Captured protobuf message to {subject}, type: {type(message).__name__}")
        except Exception as e:
            logger.error(f"Mock: Failed to serialize protobuf message to {subject}: {e}")
            raise

    # NATS callbacks not needed for mock
    async def _error_callback(self, error):
        """Mock NATS error callback."""
        pass

    async def _disconnected_callback(self):
        """Mock NATS disconnected callback."""
        pass

    async def _reconnected_callback(self):
        """Mock NATS reconnected callback."""
        pass