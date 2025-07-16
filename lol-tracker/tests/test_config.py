"""Tests for configuration module."""
import os
import pytest
from decouple import UndefinedValueError

from lol_tracker.config import Config, Environment, get_config, init_config, reset_config, is_config_initialized


def clear_all_env_vars(monkeypatch):
    """Helper to clear all environment variables that could affect config."""
    env_vars = [
        "DATABASE_URL", "RIOT_API_KEY",  # Add required vars to clear list
        "LOG_LEVEL", "LOG_FORMAT", "RIOT_API_BASE_URL", "POLL_INTERVAL_SECONDS", 
        "MESSAGE_BUS_URL", "ENVIRONMENT", "RIOT_API_REQUESTS_PER_SECOND",
        "RIOT_API_BURST_LIMIT", "RIOT_API_TIMEOUT_SECONDS", "POLL_RETRY_ATTEMPTS",
        "POLL_BACKOFF_MULTIPLIER", "POLL_MAX_BACKOFF_SECONDS", "POLL_ERROR_THRESHOLD",
        "MESSAGE_BUS_TIMEOUT_SECONDS", "MESSAGE_BUS_MAX_RECONNECT_ATTEMPTS",
        "MESSAGE_BUS_RECONNECT_DELAY_SECONDS", "TRACKING_EVENTS_SUBJECT",
        "GAME_STATE_EVENTS_SUBJECT", "CIRCUIT_BREAKER_FAILURE_THRESHOLD",
        "CIRCUIT_BREAKER_TIMEOUT_SECONDS", "CIRCUIT_BREAKER_RECOVERY_TIMEOUT_SECONDS",
        "HEALTH_CHECK_INTERVAL_SECONDS", "HEALTH_CHECK_TIMEOUT_SECONDS",
        "HEALTH_CHECK_STARTUP_DELAY_SECONDS"
    ]
    for var in env_vars:
        monkeypatch.delenv(var, raising=False)


@pytest.fixture(autouse=True)
def clean_config():
    """Reset global config before each test."""
    reset_config()
    yield
    reset_config()


def test_config_from_env_with_required_vars(monkeypatch):
    """Test creating config from environment variables."""
    clear_all_env_vars(monkeypatch)
    
    # Set required variables
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    # Set LOG_LEVEL to ensure consistent test behavior
    monkeypatch.setenv("LOG_LEVEL", "INFO")
    
    config = Config.from_env()
    
    assert config.database_url == "postgresql://test:test@localhost/test"
    assert config.riot_api_key == "test_api_key"
    assert config.riot_api_base_url == "https://na1.api.riotgames.com"
    assert config.poll_interval_seconds == 60
    assert config.log_level == "INFO"
    assert config.environment == Environment.DEVELOPMENT
    assert config.message_bus_url == "nats://localhost:4222"


def test_config_from_env_with_optional_vars(monkeypatch):
    """Test creating config with optional environment variables."""
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    monkeypatch.setenv("RIOT_API_BASE_URL", "https://euw1.api.riotgames.com")
    monkeypatch.setenv("POLL_INTERVAL_SECONDS", "30")
    monkeypatch.setenv("MESSAGE_BUS_URL", "nats://test:4222")
    monkeypatch.setenv("LOG_LEVEL", "DEBUG")
    monkeypatch.setenv("ENVIRONMENT", "production")
    
    config = Config.from_env()
    
    assert config.riot_api_base_url == "https://euw1.api.riotgames.com"
    assert config.poll_interval_seconds == 30
    assert config.message_bus_url == "nats://test:4222"
    assert config.log_level == "DEBUG"
    assert config.environment == Environment.PRODUCTION


def test_environment_validation(monkeypatch):
    """Test environment validation using Choices."""
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    monkeypatch.setenv("ENVIRONMENT", "invalid")
    
    with pytest.raises(ValueError):
        Config.from_env()


def test_log_level_validation(monkeypatch):
    """Test log level validation using Choices."""
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    monkeypatch.setenv("LOG_LEVEL", "INVALID")
    
    with pytest.raises(ValueError):
        Config.from_env()


def test_environment_helper_methods(monkeypatch):
    """Test environment helper methods."""
    clear_all_env_vars(monkeypatch)
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    
    # Test development
    monkeypatch.setenv("ENVIRONMENT", "development")
    config = Config.from_env()
    assert config.is_development()
    assert not config.is_test()
    assert not config.is_production()
    
    # Test production
    monkeypatch.setenv("ENVIRONMENT", "production")
    config = Config.from_env()
    assert not config.is_development()
    assert not config.is_test()
    assert config.is_production()
    
    # Test CI environment
    monkeypatch.setenv("ENVIRONMENT", "CI")
    config = Config.from_env()
    assert not config.is_development()
    assert config.is_test()
    assert not config.is_production()


def test_singleton_pattern(monkeypatch):
    """Test global configuration singleton pattern."""
    clear_all_env_vars(monkeypatch)
    monkeypatch.setenv("DATABASE_URL", "postgresql://test:test@localhost/test")
    monkeypatch.setenv("RIOT_API_KEY", "test_api_key")
    
    # Test not initialized
    assert not is_config_initialized()
    with pytest.raises(RuntimeError, match="Configuration not initialized"):
        get_config()
    
    # Test initialization
    config = init_config()
    assert is_config_initialized()
    assert get_config() is config
    
    # Test getting same instance
    config2 = get_config()
    assert config2 is config
    
    # Test reset
    reset_config()
    assert not is_config_initialized()
