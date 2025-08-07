"""Message bus client implementation using NATS with JetStream."""

import asyncio
import logging
from abc import ABC, abstractmethod
from typing import Optional, Protocol

import nats
from nats.aio.client import Client as NATSClient
from nats.js import JetStreamContext
import nats.js.errors


logger = logging.getLogger(__name__)


class MessageBusClient(Protocol):
    """Protocol for message bus client implementations."""

    async def connect(self) -> None:
        """Connect to the message bus."""
        ...

    async def disconnect(self) -> None:
        """Disconnect from the message bus."""
        ...

    async def create_streams(self) -> None:
        """Create required JetStream streams."""
        ...

    async def is_connected(self) -> bool:
        """Check if client is connected to the message bus."""
        ...

    async def publish(self, subject: str, data: bytes) -> None:
        """Publish a message to the specified subject."""
        ...

    async def subscribe(self, subject: str, handler) -> None:
        """Subscribe to messages on the specified subject."""
        ...


class NATSMessageBusClient:
    """NATS message bus client with JetStream support."""

    def __init__(
        self,
        servers: str,
        timeout: int = 10,
        max_reconnect_attempts: int = 10,
        reconnect_delay: int = 2,
        lol_events_stream: str = "lol_events",
        tracking_events_stream: str = "tracking_events",
        lol_events_subject: str = "lol.gamestate",
        tracking_events_subject: str = "lol.tracking",
    ):
        """Initialize NATS client.

        Args:
            servers: NATS server URLs (e.g., "nats://localhost:4222")
            timeout: Connection timeout in seconds
            max_reconnect_attempts: Maximum reconnection attempts
            reconnect_delay: Delay between reconnection attempts in seconds
            lol_events_stream: Name of the LoL events JetStream stream
            tracking_events_stream: Name of the tracking events JetStream stream
            lol_events_subject: Subject for LoL game state events
            tracking_events_subject: Subject for player tracking events
        """
        self.servers = servers
        self.timeout = timeout
        self.max_reconnect_attempts = max_reconnect_attempts
        self.reconnect_delay = reconnect_delay
        self.lol_events_stream = lol_events_stream
        self.tracking_events_stream = tracking_events_stream
        self.lol_events_subject = lol_events_subject
        self.tracking_events_subject = tracking_events_subject

        self._client: Optional[NATSClient] = None
        self._js: Optional[JetStreamContext] = None
        self._connected = False

    async def connect(self) -> None:
        """Connect to NATS server with JetStream."""
        if self._connected:
            logger.warning("Already connected to NATS")
            return

        try:
            logger.info(f"Connecting to NATS at {self.servers}")

            self._client = await nats.connect(
                servers=self.servers,
                connect_timeout=self.timeout,
                max_reconnect_attempts=self.max_reconnect_attempts,
                reconnect_time_wait=self.reconnect_delay,
                error_cb=self._error_callback,
                disconnected_cb=self._disconnected_callback,
                reconnected_cb=self._reconnected_callback,
            )

            # Enable JetStream
            self._js = self._client.jetstream()
            self._connected = True

            logger.info("Successfully connected to NATS with JetStream")

        except Exception as e:
            logger.error(f"Failed to connect to NATS: {e}")
            self._connected = False
            raise

    async def disconnect(self) -> None:
        """Disconnect from NATS server."""
        if not self._connected or not self._client:
            return

        try:
            logger.info("Disconnecting from NATS")
            await self._client.close()
            self._connected = False
            self._client = None
            self._js = None
            logger.info("Disconnected from NATS")
        except Exception as e:
            logger.error(f"Error during NATS disconnect: {e}")

    async def create_streams(self) -> None:
        """Create required JetStream streams if they don't exist."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        streams_config = [
            {
                "name": self.lol_events_stream,
                "subjects": [f"{self.lol_events_subject}.*"],
                "description": "LoL game state change events",
                "retention": "limits",
                "max_age": 24 * 60 * 60,  # 24 hours in seconds
                "max_msgs": 1000000,
                "storage": "file",
            },
            {
                "name": self.tracking_events_stream,
                "subjects": [f"{self.tracking_events_subject}.*"],
                "description": "Player tracking command events",
                "retention": "limits",
                "max_age": 24 * 60 * 60,  # 24 hours in seconds
                "max_msgs": 100000,
                "storage": "file",
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

    async def is_connected(self) -> bool:
        """Check if client is connected to NATS."""
        return (
            self._connected and self._client is not None and self._client.is_connected
        )

    async def publish(self, subject: str, data: bytes) -> None:
        """Publish a message to the specified subject using JetStream."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        try:
            ack = await self._js.publish(subject, data)
            logger.info(
                f"NATS message published - Subject: {subject}, "
                f"Size: {len(data)} bytes, Stream: {ack.stream}, Seq: {ack.seq}"
            )
        except Exception as e:
            logger.error(f"Failed to publish message to {subject}: {e}")
            raise

    async def subscribe(self, subject: str, handler) -> None:
        """Subscribe to messages on the specified subject using JetStream."""
        if not self._js:
            raise RuntimeError("Not connected to NATS JetStream")

        try:
            # Create a durable consumer for the subscription
            consumer_name = f"{subject.replace('.', '_')}_consumer"

            logger.info(
                f"Creating subscription for subject {subject} with consumer {consumer_name}"
            )

            # Subscribe with manual acknowledgment
            psub = await self._js.pull_subscribe(
                subject, consumer=consumer_name, durable=consumer_name
            )

            # Start message processing loop
            asyncio.create_task(self._message_processor(psub, handler, subject))

        except Exception as e:
            logger.error(f"Failed to subscribe to {subject}: {e}")
            raise

    async def _message_processor(self, subscription, handler, subject: str) -> None:
        """Process messages from a JetStream subscription."""
        logger.info(f"Starting message processor for subject {subject}")

        try:
            while self._connected:
                try:
                    # Fetch messages with timeout
                    msgs = await subscription.fetch(1, timeout=1.0)

                    for msg in msgs:
                        try:
                            await handler(msg.data)
                            await msg.ack()
                            logger.debug(
                                f"Processed and acknowledged message from {subject}"
                            )
                        except Exception as e:
                            logger.error(
                                f"Error processing message from {subject}: {e}"
                            )
                            await msg.nak()

                except asyncio.TimeoutError:
                    # No messages available, continue polling
                    continue
                except Exception as e:
                    logger.error(f"Error fetching messages from {subject}: {e}")
                    await asyncio.sleep(self.reconnect_delay)

        except asyncio.CancelledError:
            logger.info(f"Message processor for {subject} cancelled")
        except Exception as e:
            logger.error(f"Message processor for {subject} failed: {e}")

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

    async def __aenter__(self):
        """Async context manager entry."""
        await self.connect()
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        """Async context manager exit."""
        await self.disconnect()
