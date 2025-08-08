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
import argparse

from lol_tracker.config import Config, Environment
from lol_tracker.service import LoLTrackerService


logger = logging.getLogger(__name__)


def run_service():
    """Run the service (called by hupper in worker process)."""
    asyncio.run(main())


def start_with_reloader():
    """Start the service with hot reload using hupper."""
    try:
        import hupper
    except ImportError:
        logger.warning("hupper not installed, running without hot reload")
        run_service()
        return
    
    # hupper.start_reloader returns a reloader object in the monitor process
    # and returns None in the worker process
    reloader = hupper.start_reloader('lol_tracker.main.run_service')
    
    if reloader:
        # We're in the monitor process
        logger.info("Hot reload enabled, monitoring file changes...")
        # Watch additional files if needed
        # reloader.watch_files(['config.yaml'])


async def main(use_game_centric: bool = False):
    """Main entry point for the LoL Tracker service.
    
    Args:
        use_game_centric: Override config to use game-centric model
    """
    # Load configuration
    config = Config.from_env()
    
    # Override with CLI flag if provided
    if use_game_centric:
        config.use_game_centric_model = True

    # Set up logging
    logging.basicConfig(
        level=config.log_level,
        format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
    )

    # Set httpx and httpcore loggers to WARNING to reduce noise
    logging.getLogger("httpx").setLevel(logging.WARNING)
    logging.getLogger("httpcore").setLevel(logging.WARNING)

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
    # Parse command line arguments
    parser = argparse.ArgumentParser(description='LoL Tracker Service')
    parser.add_argument(
        '--game-centric', 
        action='store_true',
        help='Use game-centric polling model instead of state-transition model'
    )
    args = parser.parse_args()
    
    # Entry point for direct execution
    config = Config.from_env()
    
    # Enable hot reload in development
    if config.environment == Environment.DEVELOPMENT:
        # For development with reloader, we need to pass the flag differently
        # since hupper restarts the process
        if args.game_centric:
            import os
            os.environ['USE_GAME_CENTRIC_MODEL'] = 'true'
        start_with_reloader()
    else:
        asyncio.run(main(use_game_centric=args.game_centric))
