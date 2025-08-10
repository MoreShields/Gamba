"""Decorators for easy metrics instrumentation."""

import functools
import time
from typing import Callable, Any, Optional
import asyncio

from .metrics import get_metrics_provider


def measure_duration(metric_name: str, labels: Optional[dict] = None):
    """Decorator to measure function execution duration.
    
    Can be used with both sync and async functions.
    
    Args:
        metric_name: Name of the histogram metric to record to
        labels: Optional labels to add to the metric
    """
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        async def async_wrapper(*args, **kwargs) -> Any:
            metrics = get_metrics_provider()
            if not metrics:
                return await func(*args, **kwargs)
            
            start_time = time.time()
            try:
                result = await func(*args, **kwargs)
                return result
            finally:
                duration = time.time() - start_time
                # This decorator is generic, so we'd need to add support for
                # arbitrary histogram recording in MetricsProvider
                # For now, this is a placeholder
                pass
        
        @functools.wraps(func)
        def sync_wrapper(*args, **kwargs) -> Any:
            metrics = get_metrics_provider()
            if not metrics:
                return func(*args, **kwargs)
            
            start_time = time.time()
            try:
                result = func(*args, **kwargs)
                return result
            finally:
                duration = time.time() - start_time
                # This decorator is generic, so we'd need to add support for
                # arbitrary histogram recording in MetricsProvider
                # For now, this is a placeholder
                pass
        
        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper
    
    return decorator


def count_calls(metric_name: str, labels: Optional[dict] = None):
    """Decorator to count function calls.
    
    Can be used with both sync and async functions.
    
    Args:
        metric_name: Name of the counter metric to increment
        labels: Optional labels to add to the metric
    """
    def decorator(func: Callable) -> Callable:
        @functools.wraps(func)
        async def async_wrapper(*args, **kwargs) -> Any:
            metrics = get_metrics_provider()
            if metrics:
                # Would need to add generic counter support
                pass
            return await func(*args, **kwargs)
        
        @functools.wraps(func)
        def sync_wrapper(*args, **kwargs) -> Any:
            metrics = get_metrics_provider()
            if metrics:
                # Would need to add generic counter support
                pass
            return func(*args, **kwargs)
        
        if asyncio.iscoroutinefunction(func):
            return async_wrapper
        else:
            return sync_wrapper
    
    return decorator