"""Observability adapter for OpenTelemetry metrics."""

from .metrics import MetricsProvider, get_metrics_provider

__all__ = ["MetricsProvider", "get_metrics_provider"]