#!/usr/bin/env python3
"""
LoL Tracker Service - Main entry point

This service polls the Riot Games API to track League of Legends game states
and emits events to the message bus when state changes occur.
"""
import asyncio
import logging
import signal
import sys

from lol_tracker.config import Config
from lol_tracker.service import LoLTrackerService


logger = logging.getLogger(__name__)


async def main():
    """Main entry point for the LoL Tracker service."""
    # Load configuration
    config = Config.from_env()

    # Set up logging
    logging.basicConfig(
        level=config.log_level,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    # Set httpx logger to WARNING to reduce noise
    logging.getLogger("httpx").setLevel(logging.WARNING)

    logger.info("Starting LoL Tracker service")

    # Create and start the service
    service = LoLTrackerService(config)

    # Handle graceful shutdown
    def signal_handler(sig, frame):
        logger.info(f"Received signal {sig}, shutting down gracefully...")
        asyncio.create_task(service.stop())

    signal.signal(signal.SIGINT, signal_handler)
    signal.signal(signal.SIGTERM, signal_handler)

    try:
        await service.start()
    except KeyboardInterrupt:
        logger.info("Received keyboard interrupt, shutting down...")
    except Exception as e:
        logger.error(f"Service failed with error: {e}")
        sys.exit(1)
    finally:
        await service.stop()
        logger.info("LoL Tracker service stopped")


if __name__ == "__main__":
    asyncio.run(main())
