"""OpenTelemetry metrics provider for lol-tracker."""

import logging
from typing import Dict, Optional, Any
from contextlib import contextmanager
import time

from opentelemetry import metrics
from opentelemetry.sdk.metrics import MeterProvider
from opentelemetry.sdk.metrics.export import (
    ConsoleMetricExporter,
    PeriodicExportingMetricReader,
)
from opentelemetry.exporter.otlp.proto.grpc.metric_exporter import OTLPMetricExporter
from opentelemetry.sdk.resources import Resource, SERVICE_NAME

from ...config import Config
from .constants import *

logger = logging.getLogger(__name__)


class MetricsProvider:
    """Manages OpenTelemetry metrics for the lol-tracker service."""
    
    def __init__(self, config: Config):
        """Initialize the metrics provider.
        
        Args:
            config: Application configuration
        """
        self.config = config
        self._meter_provider: Optional[MeterProvider] = None
        self._meter: Optional[metrics.Meter] = None
        self._initialized = False
        
        # Metric instruments - initialized in setup()
        self._riot_api_calls_counter = None
        self._riot_api_duration_histogram = None
        self._riot_api_rate_limits_counter = None
        
        self._game_state_changes_counter = None
        self._games_detected_counter = None
        self._games_completed_counter = None
        
        self._messages_published_counter = None
        self._message_publish_failures_counter = None
        
        self._polling_iterations_counter = None
        self._polling_errors_counter = None
    
    def initialize(self) -> None:
        """Initialize the OpenTelemetry metrics provider."""
        if self._initialized:
            logger.warning("Metrics provider already initialized")
            return
        
        if not self.config.otel_enabled:
            logger.info("OpenTelemetry metrics disabled")
            self._initialized = True
            return
        
        try:
            # Create resource with service information
            resource = Resource.create({
                SERVICE_NAME: self.config.otel_service_name,
                "environment": self.config.environment.value,
            })
            
            # Create appropriate exporter based on config
            if self.config.otel_exporter_type == "console":
                exporter = ConsoleMetricExporter()
                logger.info("Using console metric exporter")
            elif self.config.otel_exporter_type == "otlp":
                exporter = OTLPMetricExporter(
                    endpoint=self.config.otel_otlp_endpoint,
                    insecure=True,  # Use insecure for local development
                )
                logger.info(f"Using OTLP metric exporter: {self.config.otel_otlp_endpoint}")
            else:
                logger.info("Metrics export disabled (exporter_type='none')")
                self._initialized = True
                return
            
            # Create metric reader with configured intervals
            reader = PeriodicExportingMetricReader(
                exporter=exporter,
                export_interval_millis=self.config.otel_export_interval_millis,
                export_timeout_millis=self.config.otel_export_timeout_millis,
            )
            
            # Create and set meter provider
            self._meter_provider = MeterProvider(
                resource=resource,
                metric_readers=[reader],
            )
            metrics.set_meter_provider(self._meter_provider)
            
            # Get meter for creating instruments
            self._meter = metrics.get_meter(__name__)
            
            # Create metric instruments
            self._create_instruments()
            
            self._initialized = True
            logger.info("Metrics provider initialized successfully")
            
        except Exception as e:
            logger.error(f"Failed to initialize metrics provider: {e}")
            self._initialized = False
            raise
    
    def _create_instruments(self) -> None:
        """Create all metric instruments."""
        if not self._meter:
            return
        
        # Riot API metrics
        self._riot_api_calls_counter = self._meter.create_counter(
            name=RIOT_API_CALLS_TOTAL,
            description="Total number of Riot API calls",
            unit="1",
        )
        
        self._riot_api_duration_histogram = self._meter.create_histogram(
            name=RIOT_API_CALL_DURATION,
            description="Duration of Riot API calls in seconds",
            unit="s",
        )
        
        self._riot_api_rate_limits_counter = self._meter.create_counter(
            name=RIOT_API_RATE_LIMITS,
            description="Total number of rate limit responses from Riot API",
            unit="1",
        )
        
        # Game state metrics
        self._game_state_changes_counter = self._meter.create_counter(
            name=GAME_STATE_CHANGES,
            description="Total number of game state changes",
            unit="1",
        )
        
        self._games_detected_counter = self._meter.create_counter(
            name=GAMES_DETECTED,
            description="Total number of games detected",
            unit="1",
        )
        
        self._games_completed_counter = self._meter.create_counter(
            name=GAMES_COMPLETED,
            description="Total number of games completed",
            unit="1",
        )
        
        # Message bus metrics
        self._messages_published_counter = self._meter.create_counter(
            name=MESSAGES_PUBLISHED,
            description="Total number of messages published to NATS",
            unit="1",
        )
        
        self._message_publish_failures_counter = self._meter.create_counter(
            name=MESSAGE_PUBLISH_FAILURES,
            description="Total number of failed message publishes",
            unit="1",
        )
        
        # Polling metrics
        self._polling_iterations_counter = self._meter.create_counter(
            name=POLLING_ITERATIONS,
            description="Total number of polling iterations",
            unit="1",
        )
        
        self._polling_errors_counter = self._meter.create_counter(
            name=POLLING_ERRORS,
            description="Total number of polling errors",
            unit="1",
        )
    
    def shutdown(self) -> None:
        """Shutdown the metrics provider and flush any pending metrics."""
        if self._meter_provider:
            try:
                self._meter_provider.shutdown()
                logger.info("Metrics provider shut down")
            except Exception as e:
                logger.error(f"Error shutting down metrics provider: {e}")
    
    # Riot API metrics
    
    def record_riot_api_call(
        self,
        endpoint_type: str,
        api_key_type: str,
        status_code: int,
        duration: float,
        error_type: Optional[str] = None
    ) -> None:
        """Record a Riot API call."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        labels = {
            LABEL_ENDPOINT_TYPE: endpoint_type,
            LABEL_API_KEY_TYPE: api_key_type,
            LABEL_STATUS_CODE: str(status_code),
        }
        
        if error_type:
            labels[LABEL_ERROR_TYPE] = error_type
        
        if self._riot_api_calls_counter:
            self._riot_api_calls_counter.add(1, labels)
        
        if self._riot_api_duration_histogram:
            self._riot_api_duration_histogram.record(duration, labels)
        
        # Track rate limits specifically
        if status_code == 429 and self._riot_api_rate_limits_counter:
            self._riot_api_rate_limits_counter.add(1, {
                LABEL_ENDPOINT_TYPE: endpoint_type,
                LABEL_API_KEY_TYPE: api_key_type,
            })
    
    @contextmanager
    def measure_riot_api_call(self, endpoint_type: str, api_key_type: str):
        """Context manager to measure Riot API call duration.
        
        Usage:
            with metrics.measure_riot_api_call("summoner", "lol") as record:
                response = await make_api_call()
                record(response.status_code)
        """
        start_time = time.time()
        
        def record(status_code: int, error_type: Optional[str] = None):
            duration = time.time() - start_time
            self.record_riot_api_call(
                endpoint_type=endpoint_type,
                api_key_type=api_key_type,
                status_code=status_code,
                duration=duration,
                error_type=error_type
            )
        
        yield record
    
    # Game state metrics
    
    def record_game_state_change(
        self,
        game_type: str,
        queue_type: Optional[str],
        transition_type: str
    ) -> None:
        """Record a game state change."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        labels = {
            LABEL_GAME_TYPE: game_type,
            LABEL_TRANSITION_TYPE: transition_type,
        }
        
        if queue_type:
            labels[LABEL_QUEUE_TYPE] = queue_type
        
        if self._game_state_changes_counter:
            self._game_state_changes_counter.add(1, labels)
    
    def record_game_detected(self, game_type: str, queue_type: Optional[str]) -> None:
        """Record a newly detected game."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        labels = {
            LABEL_GAME_TYPE: game_type,
        }
        
        if queue_type:
            labels[LABEL_QUEUE_TYPE] = queue_type
        
        if self._games_detected_counter:
            self._games_detected_counter.add(1, labels)
    
    def record_game_completed(self, game_type: str, queue_type: Optional[str]) -> None:
        """Record a completed game."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        labels = {
            LABEL_GAME_TYPE: game_type,
        }
        
        if queue_type:
            labels[LABEL_QUEUE_TYPE] = queue_type
        
        if self._games_completed_counter:
            self._games_completed_counter.add(1, labels)
    
    # Message bus metrics
    
    def record_message_published(
        self,
        event_type: str,
        subject: str,
        stream: str,
        success: bool = True
    ) -> None:
        """Record a message publish attempt."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        labels = {
            LABEL_EVENT_TYPE: event_type,
            LABEL_MESSAGE_SUBJECT: subject,
            LABEL_MESSAGE_STREAM: stream,
        }
        
        if success and self._messages_published_counter:
            self._messages_published_counter.add(1, labels)
        elif not success and self._message_publish_failures_counter:
            self._message_publish_failures_counter.add(1, labels)
    
    # Polling metrics
    
    def record_polling_iteration(self, loop_type: str) -> None:
        """Record a polling loop iteration."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        if self._polling_iterations_counter:
            self._polling_iterations_counter.add(1, {
                LABEL_LOOP_TYPE: loop_type,
            })
    
    def record_polling_error(self, loop_type: str, error_type: str) -> None:
        """Record a polling loop error."""
        if not self._initialized or not self.config.otel_enabled:
            return
        
        if self._polling_errors_counter:
            self._polling_errors_counter.add(1, {
                LABEL_LOOP_TYPE: loop_type,
                LABEL_ERROR_TYPE: error_type,
            })


# Global metrics provider instance
_metrics_provider: Optional[MetricsProvider] = None


def get_metrics_provider() -> Optional[MetricsProvider]:
    """Get the global metrics provider instance."""
    return _metrics_provider


def initialize_metrics(config: Config) -> MetricsProvider:
    """Initialize the global metrics provider.
    
    Args:
        config: Application configuration
        
    Returns:
        The initialized MetricsProvider instance
    """
    global _metrics_provider
    
    if _metrics_provider is not None:
        logger.warning("Metrics provider already initialized")
        return _metrics_provider
    
    _metrics_provider = MetricsProvider(config)
    _metrics_provider.initialize()
    
    return _metrics_provider


def shutdown_metrics() -> None:
    """Shutdown the global metrics provider."""
    global _metrics_provider
    
    if _metrics_provider:
        _metrics_provider.shutdown()
        _metrics_provider = None