"""Configuration management for LoL Tracker service."""
import os
from dataclasses import dataclass
from typing import Optional


@dataclass
class Config:
    """Configuration for the LoL Tracker service."""
    
    # Database configuration
    database_url: str
    
    # Riot API configuration
    riot_api_key: str
    riot_api_base_url: str = "https://na1.api.riotgames.com"
    
    # Polling configuration
    poll_interval_seconds: int = 60
    
    # Message bus configuration
    message_bus_url: Optional[str] = None
    
    # Logging configuration
    log_level: str = "INFO"
    
    @classmethod
    def from_env(cls) -> "Config":
        """Create configuration from environment variables."""
        database_url = os.getenv("DATABASE_URL")
        if not database_url:
            raise ValueError("DATABASE_URL environment variable is required")
        
        riot_api_key = os.getenv("RIOT_API_KEY")
        if not riot_api_key:
            raise ValueError("RIOT_API_KEY environment variable is required")
        
        return cls(
            database_url=database_url,
            riot_api_key=riot_api_key,
            riot_api_base_url=os.getenv("RIOT_API_BASE_URL", "https://na1.api.riotgames.com"),
            poll_interval_seconds=int(os.getenv("POLL_INTERVAL_SECONDS", "60")),
            message_bus_url=os.getenv("MESSAGE_BUS_URL"),
            log_level=os.getenv("LOG_LEVEL", "INFO").upper()
        )