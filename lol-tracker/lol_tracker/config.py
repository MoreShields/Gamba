"""Configuration management for LoL Tracker service."""

from dataclasses import dataclass
from enum import Enum
from typing import Optional

from decouple import Config as DecoupleConfig, Choices
from decouple import config


class Environment(Enum):
    """Supported deployment environments."""

    DEVELOPMENT = "development"
    CI = "CI"
    PRODUCTION = "production"


@dataclass
class Config:
    """Configuration for the LoL Tracker service."""

    # Required fields
    database_url: str
    database_name: str
    riot_api_key: str

    # Environment configuration
    environment: Environment = Environment.DEVELOPMENT

    # Riot API configuration
    riot_api_url: str = "https://na1.api.riotgames.com"
    riot_api_timeout_seconds: int = 30

    # Polling configuration
    poll_interval_seconds: int = 60

    # Message bus configuration (NATS)
    message_bus_url: str = "nats://localhost:4222"
    message_bus_timeout_seconds: int = 10
    message_bus_max_reconnect_attempts: int = 10
    message_bus_reconnect_delay_seconds: int = 2
    tracking_events_subject: str = "lol.tracking"
    game_state_events_subject: str = "lol.gamestate"

    # JetStream configuration
    lol_events_stream: str = "lol_events"
    tracking_events_stream: str = "tracking_events"
    jetstream_max_age_hours: int = 24
    jetstream_max_msgs_lol: int = 1000000
    jetstream_max_msgs_tracking: int = 100000
    jetstream_storage: str = "file"

    # Logging configuration
    log_level: str = "INFO"
    log_format: str = "json"

    # gRPC server configuration
    grpc_server_port: int = 50051
    grpc_server_max_workers: int = 10
    grpc_server_reflection: bool = True

    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        env = Environment(
            config(
                "ENVIRONMENT",
                default="development",
                cast=Choices(["development", "CI", "production"]),
            )
        )

        # Environment-specific defaults
        default_message_bus = (
            "nats://nats:4222"
            if env == Environment.PRODUCTION
            else "nats://localhost:4222"
        )

        return cls(
            # Required
            database_url=config("DATABASE_URL"),
            database_name=config("DATABASE_NAME", "lol_tracker_db"),
            riot_api_key=config("RIOT_API_KEY"),
            # Environment
            environment=env,
            # Riot API
            riot_api_url=config(
                "RIOT_API_URL", default="https://na1.api.riotgames.com"
            ),
            riot_api_timeout_seconds=config(
                "RIOT_API_TIMEOUT_SECONDS", default=30, cast=int
            ),
            # Polling
            poll_interval_seconds=config("POLL_INTERVAL_SECONDS", default=60, cast=int),
            # Message bus
            message_bus_url=config("MESSAGE_BUS_URL", default=default_message_bus),
            message_bus_timeout_seconds=config(
                "MESSAGE_BUS_TIMEOUT_SECONDS", default=10, cast=int
            ),
            message_bus_max_reconnect_attempts=config(
                "MESSAGE_BUS_MAX_RECONNECT_ATTEMPTS", default=10, cast=int
            ),
            message_bus_reconnect_delay_seconds=config(
                "MESSAGE_BUS_RECONNECT_DELAY_SECONDS", default=2, cast=int
            ),
            tracking_events_subject=config(
                "TRACKING_EVENTS_SUBJECT", default="lol.tracking"
            ),
            game_state_events_subject=config(
                "GAME_STATE_EVENTS_SUBJECT", default="lol.gamestate"
            ),
            # JetStream
            lol_events_stream=config("LOL_EVENTS_STREAM", default="lol_events"),
            tracking_events_stream=config(
                "TRACKING_EVENTS_STREAM", default="tracking_events"
            ),
            jetstream_max_age_hours=config(
                "JETSTREAM_MAX_AGE_HOURS", default=24, cast=int
            ),
            jetstream_max_msgs_lol=config(
                "JETSTREAM_MAX_MSGS_LOL", default=1000000, cast=int
            ),
            jetstream_max_msgs_tracking=config(
                "JETSTREAM_MAX_MSGS_TRACKING", default=100000, cast=int
            ),
            jetstream_storage=config("JETSTREAM_STORAGE", default="file"),
            # Logging
            log_level=config(
                "LOG_LEVEL",
                default="INFO",
                cast=lambda x: Choices(
                    ["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"]
                )(x.upper()),
            ),
            log_format=config(
                "LOG_FORMAT",
                default="json",
                cast=lambda x: Choices(["json", "text"])(x.lower()),
            ),
            # gRPC server
            grpc_server_port=config("GRPC_SERVER_PORT", default=9000, cast=int),
            grpc_server_max_workers=config(
                "GRPC_SERVER_MAX_WORKERS", default=10, cast=int
            ),
            grpc_server_reflection=config(
                "GRPC_SERVER_REFLECTION", default=True, cast=bool
            ),
        )

    def is_development(self) -> bool:
        """Check if running in development environment."""
        return self.environment == Environment.DEVELOPMENT

    def is_test(self) -> bool:
        """Check if running in test environment."""
        return self.environment == Environment.CI

    def is_production(self) -> bool:
        """Check if running in production environment."""
        return self.environment == Environment.PRODUCTION

    def get_database_url(self) -> str:
        """Construct the full database URL by combining base URL and database name."""
        from urllib.parse import urlparse, urlunparse

        # Parse the database URL
        parsed = urlparse(self.database_url)

        # Ensure we have the asyncpg driver specified
        scheme = parsed.scheme
        if scheme == "postgres":
            # Convert legacy postgres:// to postgresql://
            scheme = "postgresql+asyncpg"
        elif scheme == "postgresql":
            scheme = "postgresql+asyncpg"

        # Replace the database name (path component)
        # The path includes the leading '/', so we prepend it to database_name
        path = f"/{self.database_name}"

        # Reconstruct the URL with the new scheme and database name
        return urlunparse(
            (scheme, parsed.netloc, path, parsed.params, parsed.query, parsed.fragment)
        )
