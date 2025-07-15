"""Tests for configuration module."""
import os
import pytest

from lol_tracker.config import Config


def test_config_from_env_with_required_vars(monkeypatch):
    """Test creating config from environment variables."""
    # Clear all optional environment variables
    monkeypatch.delenv("LOG_LEVEL", raising=False)
    monkeypatch.delenv("RIOT_API_BASE_URL", raising=False)
    monkeypatch.delenv("POLL_INTERVAL_SECONDS", raising=False)
    monkeypatch.delenv("MESSAGE_BUS_URL", raising=False)
    
    # Set required variables
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    
    config = Config.from_env()
    
    assert config.database_url == "postgresql://test:test@localhost/test"
    assert config.riot_api_key == "test_api_key"
    assert config.riot_api_base_url == "https://na1.api.riotgames.com"
    assert config.poll_interval_seconds == 60
    assert config.log_level == "INFO"


def test_config_from_env_missing_database_url():
    """Test that missing DATABASE_URL raises ValueError."""
    # Clear environment
    if "DATABASE_URL" in os.environ:
        del os.environ["DATABASE_URL"]
    
    with pytest.raises(ValueError, match="DATABASE_URL environment variable is required"):
        Config.from_env()


def test_config_from_env_missing_riot_api_key(monkeypatch):
    """Test that missing RIOT_API_KEY raises ValueError."""
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.delenv("RIOT_API_KEY", raising=False)
    
    with pytest.raises(ValueError, match="RIOT_API_KEY environment variable is required"):
        Config.from_env()


def test_config_from_env_with_optional_vars(monkeypatch):
    """Test creating config with optional environment variables."""
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    monkeypatch.setenv("RIOT_API_BASE_URL", "https://euw1.api.riotgames.com")
    monkeypatch.setenv("POLL_INTERVAL_SECONDS", "30")
    monkeypatch.setenv("MESSAGE_BUS_URL", "amqp://localhost")
    monkeypatch.setenv("LOG_LEVEL", "DEBUG")
    
    config = Config.from_env()
    
    assert config.riot_api_base_url == "https://euw1.api.riotgames.com"
    assert config.poll_interval_seconds == 30
    assert config.message_bus_url == "amqp://localhost"
    assert config.log_level == "DEBUG"