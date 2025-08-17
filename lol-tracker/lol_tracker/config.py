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
    tft_riot_api_key: str  # TFT-specific API key

    # Environment configuration
    environment: Environment = Environment.DEVELOPMENT

    # Riot API configuration
    riot_api_url: str = "https://na1.api.riotgames.com"
    riot_api_timeout_seconds: int = 30

    # Polling configuration
    poll_interval_seconds: int = 60
    
    # Game-centric polling intervals
    detection_interval_seconds: int = 30
    completion_interval_seconds: int = 60

    # Message bus configuration (NATS)
    message_bus_url: str = "nats://localhost:4222"
    message_bus_timeout_seconds: int = 10
    message_bus_max_reconnect_attempts: int = 10
    message_bus_reconnect_delay_seconds: int = 2
    tracking_events_subject: str = "lol.tracking"
    game_state_events_subject: str = "lol.gamestate"
    tft_game_state_events_subject: str = "tft.gamestate"

    # JetStream configuration
    lol_events_stream: str = "lol_events"
    tft_events_stream: str = "tft_events"
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
        # Helper function to simplify config calls
        def get_config(key: str, default=None, cast=None):
            if default is not None:
                if cast is not None:
                    return config(key, default=default, cast=cast)
                else:
                    return config(key, default=default)
            else:
                return config(key)
        
        env = Environment(
            get_config("ENVIRONMENT", "development", Choices(["development", "CI", "production"]))
        )

        # Environment-specific defaults
        default_message_bus = "nats://nats:4222" if env == Environment.PRODUCTION else "nats://localhost:4222"

        return cls(
            # Required
            database_url=get_config("DATABASE_URL"),
            database_name=get_config("DATABASE_NAME", "lol_tracker_db"),
            riot_api_key=get_config("RIOT_API_KEY"),
            tft_riot_api_key=get_config("TFT_RIOT_API_KEY"),
            # Environment
            environment=env,
            # Riot API
            riot_api_url=get_config("RIOT_API_URL", "https://na1.api.riotgames.com"),
            riot_api_timeout_seconds=get_config("RIOT_API_TIMEOUT_SECONDS", 30, int),
            # Polling
            poll_interval_seconds=get_config("POLL_INTERVAL_SECONDS", 60, int),
            # Game-centric polling intervals
            detection_interval_seconds=get_config("DETECTION_INTERVAL_SECONDS", 30, int),
            completion_interval_seconds=get_config("COMPLETION_INTERVAL_SECONDS", 60, int),
            # Message bus
            message_bus_url=get_config("MESSAGE_BUS_URL", default_message_bus),
            message_bus_timeout_seconds=get_config("MESSAGE_BUS_TIMEOUT_SECONDS", 10, int),
            message_bus_max_reconnect_attempts=get_config("MESSAGE_BUS_MAX_RECONNECT_ATTEMPTS", 10, int),
            message_bus_reconnect_delay_seconds=get_config("MESSAGE_BUS_RECONNECT_DELAY_SECONDS", 2, int),
            tracking_events_subject=get_config("TRACKING_EVENTS_SUBJECT", "lol.tracking"),
            game_state_events_subject=get_config("GAME_STATE_EVENTS_SUBJECT", "lol.gamestate"),
            tft_game_state_events_subject=get_config("TFT_GAME_STATE_EVENTS_SUBJECT", "tft.gamestate"),
            # JetStream
            lol_events_stream=get_config("LOL_EVENTS_STREAM", "lol_events"),
            tft_events_stream=get_config("TFT_EVENTS_STREAM", "tft_events"),
            tracking_events_stream=get_config("TRACKING_EVENTS_STREAM", "tracking_events"),
            jetstream_max_age_hours=get_config("JETSTREAM_MAX_AGE_HOURS", 24, int),
            jetstream_max_msgs_lol=get_config("JETSTREAM_MAX_MSGS_LOL", 1000000, int),
            jetstream_max_msgs_tracking=get_config("JETSTREAM_MAX_MSGS_TRACKING", 100000, int),
            jetstream_storage=get_config("JETSTREAM_STORAGE", "file"),
            # Logging
            log_level=get_config("LOG_LEVEL", "INFO"),
            log_format=get_config("LOG_FORMAT", "json", Choices(["json", "text"])),
            # gRPC server
            grpc_server_port=get_config("GRPC_SERVER_PORT", 9000, int),
            grpc_server_max_workers=get_config("GRPC_SERVER_MAX_WORKERS", 10, int),
            grpc_server_reflection=get_config("GRPC_SERVER_REFLECTION", True, bool),
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

    def get_tft_api_key(self) -> str:
        """Get the TFT API key."""
        return self.tft_riot_api_key

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
