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
    riot_api_base_url: str = "https://na1.api.riotgames.com"
    riot_api_requests_per_second: int = 20
    riot_api_burst_limit: int = 100
    riot_api_timeout_seconds: int = 30
    
    # Polling configuration
    poll_interval_seconds: int = 60
    poll_retry_attempts: int = 3
    poll_backoff_multiplier: float = 2.0
    poll_max_backoff_seconds: int = 300
    poll_error_threshold: int = 5
    
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
    
    # Circuit breaker configuration
    circuit_breaker_failure_threshold: int = 5
    circuit_breaker_timeout_seconds: int = 60
    circuit_breaker_recovery_timeout_seconds: int = 30
    
    # Health check configuration
    health_check_interval_seconds: int = 30
    health_check_timeout_seconds: int = 10
    health_check_startup_delay_seconds: int = 30
    
    # Logging configuration
    log_level: str = "INFO"
    log_format: str = "json"

    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        env = Environment(config("ENVIRONMENT", default="development", cast=Choices(["development", "CI", "production"])))
        
        # Environment-specific defaults
        default_message_bus = "nats://nats:4222" if env == Environment.PRODUCTION else "nats://localhost:4222"
        
        return cls(
            # Required
            database_url=config("DATABASE_URL"),
            database_name=config("DATABASE_NAME", "lol_tracker_db"),
            riot_api_key=config("RIOT_API_KEY"),
            
            # Environment
            environment=env,
            
            # Riot API
            riot_api_base_url=config("RIOT_API_BASE_URL", default="https://na1.api.riotgames.com"),
            riot_api_requests_per_second=config("RIOT_API_REQUESTS_PER_SECOND", default=20, cast=int),
            riot_api_burst_limit=config("RIOT_API_BURST_LIMIT", default=100, cast=int),
            riot_api_timeout_seconds=config("RIOT_API_TIMEOUT_SECONDS", default=30, cast=int),
            
            # Polling
            poll_interval_seconds=config("POLL_INTERVAL_SECONDS", default=60, cast=int),
            poll_retry_attempts=config("POLL_RETRY_ATTEMPTS", default=3, cast=int),
            poll_backoff_multiplier=config("POLL_BACKOFF_MULTIPLIER", default=2.0, cast=float),
            poll_max_backoff_seconds=config("POLL_MAX_BACKOFF_SECONDS", default=300, cast=int),
            poll_error_threshold=config("POLL_ERROR_THRESHOLD", default=5, cast=int),
            
            # Message bus
            message_bus_url=config("MESSAGE_BUS_URL", default=default_message_bus),
            message_bus_timeout_seconds=config("MESSAGE_BUS_TIMEOUT_SECONDS", default=10, cast=int),
            message_bus_max_reconnect_attempts=config("MESSAGE_BUS_MAX_RECONNECT_ATTEMPTS", default=10, cast=int),
            message_bus_reconnect_delay_seconds=config("MESSAGE_BUS_RECONNECT_DELAY_SECONDS", default=2, cast=int),
            tracking_events_subject=config("TRACKING_EVENTS_SUBJECT", default="lol.tracking"),
            game_state_events_subject=config("GAME_STATE_EVENTS_SUBJECT", default="lol.gamestate"),
            
            # JetStream
            lol_events_stream=config("LOL_EVENTS_STREAM", default="lol_events"),
            tracking_events_stream=config("TRACKING_EVENTS_STREAM", default="tracking_events"),
            jetstream_max_age_hours=config("JETSTREAM_MAX_AGE_HOURS", default=24, cast=int),
            jetstream_max_msgs_lol=config("JETSTREAM_MAX_MSGS_LOL", default=1000000, cast=int),
            jetstream_max_msgs_tracking=config("JETSTREAM_MAX_MSGS_TRACKING", default=100000, cast=int),
            jetstream_storage=config("JETSTREAM_STORAGE", default="file"),
            
            # Circuit breaker
            circuit_breaker_failure_threshold=config("CIRCUIT_BREAKER_FAILURE_THRESHOLD", default=5, cast=int),
            circuit_breaker_timeout_seconds=config("CIRCUIT_BREAKER_TIMEOUT_SECONDS", default=60, cast=int),
            circuit_breaker_recovery_timeout_seconds=config("CIRCUIT_BREAKER_RECOVERY_TIMEOUT_SECONDS", default=30, cast=int),
            
            # Health check
            health_check_interval_seconds=config("HEALTH_CHECK_INTERVAL_SECONDS", default=30, cast=int),
            health_check_timeout_seconds=config("HEALTH_CHECK_TIMEOUT_SECONDS", default=10, cast=int),
            health_check_startup_delay_seconds=config("HEALTH_CHECK_STARTUP_DELAY_SECONDS", default=30, cast=int),
            
            # Logging
            log_level=config("LOG_LEVEL", default="INFO", cast=lambda x: Choices(["DEBUG", "INFO", "WARNING", "ERROR", "CRITICAL"])(x.upper())),
            log_format=config("LOG_FORMAT", default="json", cast=lambda x: Choices(["json", "text"])(x.lower()))
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
        if scheme == 'postgres':
            # Convert legacy postgres:// to postgresql://
            scheme = 'postgresql+asyncpg'
        elif scheme == 'postgresql':
            scheme = 'postgresql+asyncpg'
        
        # Replace the database name (path component)
        # The path includes the leading '/', so we prepend it to database_name
        path = f"/{self.database_name}"
        
        # Reconstruct the URL with the new scheme and database name
        return urlunparse((
            scheme,
            parsed.netloc,
            path,
            parsed.params,
            parsed.query,
            parsed.fragment
        ))


# Global configuration instance management (singleton pattern)
_config_instance: Optional[Config] = None


def get_config() -> Config:
    """Get the global configuration instance.
    
    Returns:
        Config: The global configuration instance
        
    Raises:
        RuntimeError: If configuration has not been initialized
    """
    global _config_instance
    if _config_instance is None:
        raise RuntimeError(
            "Configuration not initialized. Call init_config() first or use Config.from_env() directly."
        )
    return _config_instance


def init_config(config_instance: Config = None) -> Config:
    """Initialize the global configuration instance.
    
    Args:
        config_instance: Optional Config instance. If not provided, will load from environment.
        
    Returns:
        Config: The initialized configuration instance
    """
    global _config_instance
    if config_instance is None:
        config_instance = Config.from_env()
    
    _config_instance = config_instance
    return _config_instance


def reset_config() -> None:
    """Reset the global configuration instance.
    
    This is primarily useful for testing.
    """
    global _config_instance
    _config_instance = None


def is_config_initialized() -> bool:
    """Check if the global configuration has been initialized.
    
    Returns:
        bool: True if configuration is initialized, False otherwise
    """
    return _config_instance is not None