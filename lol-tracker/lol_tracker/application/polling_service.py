"""Consolidated polling service for the LoL Tracker application.

This service consolidates the functionality of multiple specialized services
into a single, pragmatic implementation that handles:
- Game state polling and management
- Match result processing
- Task scheduling using asyncio directly
"""

import logging
import asyncio
from typing import List, Optional, Dict, Any

from ..core.entities import Player, GameState
from ..core.enums import GameStatus, QueueType
from ..core.services import GameStateTransitionService
from ..adapters.database.manager import DatabaseManager
from ..adapters.riot_api.client import RiotAPIClient, PlayerNotInGameError
from ..adapters.messaging.events import EventPublisher
from ..config import Config


logger = logging.getLogger(__name__)




class PollingService:
    """Consolidated polling service that handles all game state polling operations.
    
    This service provides a single, pragmatic interface for:
    - Game state polling and management
    - Match result processing 
    - Task scheduling using asyncio directly
    """
    
    def __init__(
        self, 
        database: DatabaseManager,
        riot_api: RiotAPIClient,
        event_publisher: EventPublisher,
        config: Config
    ):
        """Initialize the consolidated polling service.
        
        Args:
            database: Database manager for data operations
            riot_api: Riot API client for game data
            event_publisher: Event publisher for domain events
            config: Application configuration
        """
        self.database = database
        self.riot_api = riot_api
        self.event_publisher = event_publisher
        self.config = config
        
        # Initialize domain services
        self.game_state_transition_service = GameStateTransitionService()
        
        # Extract frequently used values
        self.poll_interval_seconds = config.poll_interval_seconds
        
        # Polling state
        self._is_running = False
        self._polling_task: Optional[asyncio.Task] = None
    
    # Core polling operations
    
    async def start_polling(self) -> None:
        """Start the game state polling loop."""
        if self._is_running:
            logger.warning("Polling is already running")
            return
        
        self._is_running = True
        self._polling_task = asyncio.create_task(self._polling_loop())
        
        logger.info(
            f"Started game state polling with {self.poll_interval_seconds}s intervals"
        )
    
    async def stop_polling(self) -> None:
        """Stop the game state polling loop."""
        if not self._is_running:
            logger.warning("Polling is not running")
            return
        
        self._is_running = False
        
        # Cancel main polling task
        if self._polling_task and not self._polling_task.done():
            self._polling_task.cancel()
            try:
                await self._polling_task
            except asyncio.CancelledError:
                pass
        
        
        logger.info("Stopped game state polling")
    
    async def poll_once(self) -> List[GameState]:
        """Execute a single polling cycle.
        
        Returns:
            List of game state updates detected
        """
        try:
            state_updates = await self._execute_poll_game_states()
            
            # Match results are now processed immediately during state transition
            
            if state_updates:
                logger.info(f"Poll cycle detected {len(state_updates)} state changes")
            
            return state_updates
            
        except Exception as e:
            logger.error(f"Error in polling cycle: {e}")
            raise
    
    # Private implementation methods
    
    async def _polling_loop(self) -> None:
        """Main polling loop that runs continuously."""
        logger.info("Game state polling loop started")
        
        while self._is_running:
            try:
                await self.poll_once()
                
                
                # Wait for next poll cycle
                await asyncio.sleep(self.poll_interval_seconds)
                
            except asyncio.CancelledError:
                logger.info("Polling loop cancelled")
                break
            except Exception as e:
                logger.error(f"Error in polling loop: {e}")
                # Wait before retrying on error
                await asyncio.sleep(min(self.poll_interval_seconds, 30))
        
        logger.info("Game state polling loop stopped")
    
    async def _execute_poll_game_states(self) -> List[GameState]:
        """Execute the poll game states logic.
        
        Returns:
            List of game state updates that occurred
        """
        logger.debug("Polling game states")
        state_updates = []
        
        # Get all active tracked players
        active_players = await self.database.get_all_players()
        
        if not active_players:
            logger.debug("No active players to poll")
            return state_updates
        
        logger.debug(f"Polling game states for {len(active_players)} active players")
        
        # Track successful polls
        successful_polls = 0
        failed_polls = 0
        
        # Poll each player's game state
        for player in active_players:
            try:
                update = await self._poll_single_player(player)
                if update:
                    state_updates.append(update)
                successful_polls += 1
            except Exception as e:
                logger.error(f"Error polling player {player.game_name}#{player.tag_line}: {e}")
                failed_polls += 1
                # Continue with other players even if one fails
                continue
        
        # Log polling summary
        logger.info(
            f"Polling cycle complete: {successful_polls}/{len(active_players)} players polled successfully, "
            f"{len(state_updates)} state changes detected"
        )
        if failed_polls > 0:
            logger.warning(f"{failed_polls} players failed to poll in this cycle")
        
        return state_updates
    
    async def _poll_single_player(
        self, 
        player: Player
    ) -> Optional[GameState]:
        """Poll a single player's game state and detect changes.
        
        Args:
            player: The player to poll
            
        Returns:
            GameState if state changed, None otherwise
        """
        if not player.can_be_tracked():
            logger.warning(f"Player {player.riot_id} cannot be tracked, skipping")
            return None
        
        if player.id is None:
            logger.error(f"Player {player.riot_id} has no ID, cannot track")
            return None
        
        # Ensure player has a game state
        current_state = await self._ensure_player_has_game_state(player)
        
        # Get current state from Riot API
        api_response = await self._get_riot_game_info(player)
        
        # First check if state would change using domain service
        potential_new_state, state_changed = self.game_state_transition_service.handle_riot_api_response(
            player,
            current_state,
            api_response
        )
        
        if not state_changed:
            return None
        
        # If this is a game end transition, fetch match results BEFORE saving state
        if (current_state.status == GameStatus.IN_GAME and 
            potential_new_state.status == GameStatus.NOT_IN_GAME and 
            current_state.game_id):
            try:
                # Fetch match details - REQUIRED for game end transitions
                match_info = await self.riot_api.get_match_for_game(
                    current_state.game_id, 
                    current_state.queue_type,
                    region="na1"
                )
                if not match_info:
                    logger.error(f"Failed to fetch match details for game {current_state.game_id} - aborting state transition")
                    return None  # Don't commit state change without match results
                
                # Update the state with match results
                game_result = potential_new_state.update_from_match_info(
                    match_info, player.game_name, player.tag_line
                )
                if game_result:
                    logger.info(
                        f"Fetched match result for {player.riot_id}: {game_result}"
                    )
                else:
                    logger.warning(f"Could not process match result for player {player.riot_id} - aborting state transition")
                    return None  # Don't commit if we can't process the match
                    
            except Exception as e:
                logger.error(f"Error fetching match results for {current_state.game_id}: {e} - aborting state transition")
                return None  # Don't commit state change on API errors
        
        # Now save the state to database
        await self.database.create_game_state(
            player_id=potential_new_state.player_id,
            status=potential_new_state.status.value,
            game_id=potential_new_state.game_id,
            queue_type=potential_new_state.queue_type.value if potential_new_state.queue_type else None,
            game_start_time=potential_new_state.game_start_time,
            raw_api_response=str(api_response) if api_response else None
        )
        
        # Log the state change
        logger.info(
            f"Game state changed for {player.riot_id}: "
            f"{current_state.status.value} -> {potential_new_state.status.value}"
        )
        
        # Use the updated state as the new state
        new_state = potential_new_state
        
        # Publish event for the state change
        try:
            if player.id:
                # Let the domain create the appropriate event
                event = new_state.create_state_change_event(player, current_state)
                await self.event_publisher.publish_game_state_changed(event)
                logger.debug(f"Published {event.get_event_type()} event for player {player.id}")
        except Exception as e:
            logger.error(f"Failed to publish event for player {player.riot_id}: {e}")
            # Don't re-raise to avoid breaking the main flow
        
        return new_state
    
    async def _ensure_player_has_game_state(self, player: Player) -> GameState:
        """Ensure player has a game state, creating initial state if needed.
        
        Args:
            player: The player to check
            
        Returns:
            Current GameState for the player
        """
        current_state_record = await self.database.get_latest_game_state_for_player(player.id)
        if not current_state_record:
            # Create initial state
            current_state = GameState.create_not_in_game(player.id)
            current_state_record = await self.database.create_game_state(
                player_id=player.id,
                status=current_state.status.value,
                game_id=current_state.game_id,
                queue_type=current_state.queue_type.value if current_state.queue_type else None,
                game_start_time=current_state.game_start_time
            )
            logger.debug(f"Created initial game state for player {player.riot_id}")
        
        return current_state_record
    
    async def _get_riot_game_info(self, player: Player) -> Optional[Dict[str, Any]]:
        """Get current game information from Riot API.
        
        Args:
            player: The player to query
            
        Returns:
            Raw API response dict if player is in game, None otherwise
        """
        try:
            # Use the parallel checking method that tries both LoL and TFT
            current_game = await self.riot_api.get_active_game_info(player.game_name, player.tag_line)
            if current_game:
                # Use the polymorphic to_dict method
                return current_game.to_dict()
            return None
        except PlayerNotInGameError:
            # Expected when player is not in game
            return None
    
