"""Main service class for LoL Tracker."""
import asyncio
import logging
from typing import Optional

from lol_tracker.config import Config


logger = logging.getLogger(__name__)


class LoLTrackerService:
    """Main service class that orchestrates the LoL tracking functionality."""
    
    def __init__(self, config: Config):
        self.config = config
        self._running = False
        self._tasks: list[asyncio.Task] = []
    
    async def start(self):
        """Start the LoL Tracker service."""
        logger.info("Starting LoL Tracker service")
        self._running = True
        
        # TODO: Initialize database connection
        # TODO: Initialize message bus connection
        # TODO: Initialize Riot API client
        # TODO: Start polling loop
        
        # For now, just run a simple loop
        while self._running:
            logger.info("LoL Tracker service running...")
            await asyncio.sleep(self.config.poll_interval_seconds)
    
    async def stop(self):
        """Stop the LoL Tracker service."""
        logger.info("Stopping LoL Tracker service")
        self._running = False
        
        # Cancel all running tasks
        for task in self._tasks:
            task.cancel()
        
        # Wait for tasks to complete
        if self._tasks:
            await asyncio.gather(*self._tasks, return_exceptions=True)
        
        # TODO: Close database connection
        # TODO: Close message bus connection
        
        logger.info("LoL Tracker service stopped")