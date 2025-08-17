"""Game-centric polling service for the LoL Tracker application.

This service implements the new game-centric architecture with two independent
polling loops: one for detecting new games and one for completing active games.
This eliminates race conditions and simplifies the overall tracking logic.
"""

import logging
import asyncio
from typing import List, Optional, Dict, Any
from datetime import datetime

from ..core.entities import Player, TrackedGame, LoLGameResult, TFTGameResult
from ..core.enums import GameStatus, QueueType
from ..core.events import GameStateChangedEvent
from ..adapters.database.manager import DatabaseManager
from ..adapters.riot_api.client import RiotAPIClient, PlayerNotInGameError
from ..adapters.messaging.events import EventPublisher
from ..config import Config


logger = logging.getLogger(__name__)


class GameCentricPollingService:
    """Game-centric polling service with two independent loops.
    
    This service provides a cleaner, more reliable approach to game tracking:
    - Detection loop: Discovers new games and creates tracked_games entries
    - Completion loop: Monitors active games and fetches results when complete
    
    Each game is tracked as a single row that progresses through its lifecycle,
    eliminating the race conditions inherent in state-transition models.
    """
    
    def __init__(
        self,
        database: DatabaseManager,
        riot_api: RiotAPIClient,
        event_publisher: EventPublisher,
        config: Config
    ):
        """Initialize the game-centric polling service.
        
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
        
        # Extract polling intervals from config
        self.detection_interval = getattr(config, 'detection_interval_seconds', 30)
        self.completion_interval = getattr(config, 'completion_interval_seconds', 60)
        
        # Polling state
        self._is_running = False
        self._detection_task: Optional[asyncio.Task] = None
        self._completion_task: Optional[asyncio.Task] = None
    
    # Event Creation Helpers
    
    def _create_game_start_event(self, player: Player, game: TrackedGame) -> GameStateChangedEvent:
        """Create event for game start."""
        from ..core.events import LoLGameStateChangedEvent, TFTGameStateChangedEvent
        
        common_kwargs = {
            'player_id': player.id,
            'game_name': player.game_name,
            'tag_line': player.tag_line,
            'previous_status': 'NOT_IN_GAME',
            'new_status': 'IN_GAME',
            'game_id': game.game_id,
            'queue_type': game.queue_type.value if game.queue_type else None,
            'changed_at': datetime.utcnow(),
            'is_game_start': True,
            'is_game_end': False,
            'duration_seconds': None
        }
        
        # Create appropriate event type based on game type
        if game.game_type == 'TFT':
            return TFTGameStateChangedEvent(**common_kwargs)
        else:
            return LoLGameStateChangedEvent(**common_kwargs)
    
    def _create_game_end_event(self, player: Player, game: TrackedGame) -> Optional[GameStateChangedEvent]:
        """Create event for game end.
        Returns None if no game result available."""
        if not game.game_result:
            return None  # Critical: no event without results
        
        from ..core.events import LoLGameStateChangedEvent, TFTGameStateChangedEvent
        
        common_kwargs = {
            'player_id': player.id,
            'game_name': player.game_name,
            'tag_line': player.tag_line,
            'previous_status': 'IN_GAME',
            'new_status': 'NOT_IN_GAME',
            'game_id': game.game_id,
            'queue_type': game.queue_type.value if game.queue_type else None,
            'changed_at': datetime.utcnow(),
            'is_game_start': False,
            'is_game_end': True,
            'duration_seconds': game.duration_seconds
        }
        
        # Create appropriate event type based on game type
        if game.game_type == 'TFT':
            return TFTGameStateChangedEvent(
                **common_kwargs,
                placement=game.game_result.placement if isinstance(game.game_result, TFTGameResult) else None
            )
        else:
            return LoLGameStateChangedEvent(
                **common_kwargs,
                won=game.game_result.won if isinstance(game.game_result, LoLGameResult) else None,
                champion_played=game.game_result.champion_played if isinstance(game.game_result, LoLGameResult) else None
            )
    
    # Public API
    
    async def start_polling(self) -> None:
        """Start both polling loops."""
        if self._is_running:
            logger.warning("Game-centric polling is already running")
            return
        
        self._is_running = True
        
        # Start both loops concurrently
        self._detection_task = asyncio.create_task(self._detection_loop())
        self._completion_task = asyncio.create_task(self._completion_loop())
        
        logger.info(
            f"Started game-centric polling - Detection: {self.detection_interval}s, "
            f"Completion: {self.completion_interval}s"
        )
    
    async def stop_polling(self) -> None:
        """Stop both polling loops gracefully."""
        if not self._is_running:
            logger.warning("Game-centric polling is not running")
            return
        
        self._is_running = False
        
        # Cancel both tasks
        tasks_to_cancel = []
        if self._detection_task and not self._detection_task.done():
            tasks_to_cancel.append(self._detection_task)
        if self._completion_task and not self._completion_task.done():
            tasks_to_cancel.append(self._completion_task)
        
        for task in tasks_to_cancel:
            task.cancel()
            try:
                await task
            except asyncio.CancelledError:
                pass
        
        logger.info("Stopped game-centric polling")
    
    # Detection Loop - Find new games
    
    async def _detection_loop(self) -> None:
        """Main detection loop that discovers new games."""
        logger.info("Game detection loop started")
        
        while self._is_running:
            try:
                await self._detect_new_games()
                await asyncio.sleep(self.detection_interval)
                
            except asyncio.CancelledError:
                logger.info("Detection loop cancelled")
                break
            except Exception as e:
                logger.error(f"Error in detection loop: {e}")
                # Wait before retrying on error
                await asyncio.sleep(min(self.detection_interval, 30))
        
        logger.info("Game detection loop stopped")
    
    async def _detect_new_games(self) -> None:
        """Detect new games for all tracked players."""
        logger.debug("Detecting new games")
        
        # Get all tracked players
        players = await self.database.get_all_players()
        if not players:
            logger.debug("No tracked players")
            return
        
        logger.debug(f"Checking {len(players)} players for new games")
        
        new_games_detected = 0
        for player in players:
            try:
                if await self._detect_game_for_player(player):
                    new_games_detected += 1
            except Exception as e:
                logger.error(f"Error detecting game for {player.game_name}#{player.tag_line}: {e}")
                continue
        
        if new_games_detected > 0:
            logger.info(f"Detected {new_games_detected} new games")
    
    async def _detect_game_for_player(self, player: Player) -> bool:
        """Detect if a player has started a new game.
        
        Args:
            player: The player to check
            
        Returns:
            True if a new game was detected and created, False otherwise
        """
        if not player.can_be_tracked() or player.id is None:
            return False
        
        # Check if player is currently in a game
        try:
            current_game = await self.riot_api.get_active_game_info(
                player.game_name, 
                player.tag_line
            )
            if not current_game:
                return False
                
            game_data = current_game.to_dict()
            game_id = game_data.get('gameId')
            
            if not game_id:
                logger.warning(f"No game ID in API response for {player.riot_id}")
                return False
            
            # Check if we're already tracking this game
            existing_game = await self.database.get_tracked_game(player.id, str(game_id))
            if existing_game:
                logger.debug(f"Game {game_id} already tracked for {player.riot_id}")
                return False
            
            # Create new tracked game entry
            queue_id = game_data.get('gameQueueConfigId')
            queue_type = QueueType.from_queue_id(queue_id) if queue_id else None
            
            # Extract game_type from API response
            game_type = game_data.get('game_type', 'LOL')  # Default to LOL if not specified
            
            tracked_game_model = await self.database.create_tracked_game(
                player_id=player.id,
                game_id=str(game_id),
                game_type=game_type,
                status='ACTIVE',
                queue_type=queue_type.value if queue_type else None,
                started_at=datetime.utcnow(),
                raw_api_response=str(game_data)
            )
            
            logger.info(
                f"Detected new game {game_id} for {player.riot_id} "
                f"(Queue: {queue_type.value if queue_type else 'Unknown'})"
            )
            
            # Create domain entity for event creation
            tracked_game_entity = TrackedGame(
                player_id=player.id,
                game_id=str(game_id),
                game_type=game_type,
                status='ACTIVE',
                detected_at=tracked_game_model.detected_at,
                started_at=tracked_game_model.started_at,
                queue_type=queue_type,
                id=tracked_game_model.id
            )
            
            # Emit game started event using proper event object
            try:
                event = self._create_game_start_event(player, tracked_game_entity)
                await self.event_publisher.publish_game_state_changed(event)
                logger.debug(f"Published {event.get_event_type()} event for game {game_id}")
            except Exception as e:
                logger.error(f"Failed to publish game started event: {e}")
            
            return True
            
        except PlayerNotInGameError:
            # Expected when player is not in game
            return False
    
    # Completion Loop - Monitor and complete active games
    
    async def _completion_loop(self) -> None:
        """Main completion loop that monitors active games."""
        logger.info("Game completion loop started")
        
        # Wait a bit before starting to avoid race with detection loop
        await asyncio.sleep(5)
        
        while self._is_running:
            try:
                await self._check_active_games()
                await asyncio.sleep(self.completion_interval)
                
            except asyncio.CancelledError:
                logger.info("Completion loop cancelled")
                break
            except Exception as e:
                logger.error(f"Error in completion loop: {e}")
                # Wait before retrying on error
                await asyncio.sleep(min(self.completion_interval, 30))
        
        logger.info("Game completion loop stopped")
    
    async def _check_active_games(self) -> None:
        """Check all active games for completion."""
        logger.debug("Checking active games for completion")
        
        # Get all active games
        active_games = await self.database.get_games_by_status('ACTIVE')
        if not active_games:
            logger.debug("No active games to check")
            return
        
        logger.debug(f"Checking {len(active_games)} active games")
        
        completed_games = 0
        for game in active_games:
            try:
                if await self._check_game_completion(game):
                    completed_games += 1
            except Exception as e:
                logger.error(f"Error checking game {game.game_id}: {e}")
                # Update error info but keep game active for retry
                await self.database.update_game_error(
                    game.player_id,
                    game.game_id,
                    str(e)
                )
                continue
        
        if completed_games > 0:
            logger.info(f"Completed {completed_games} games")
    
    async def _check_game_completion(self, game) -> bool:
        """Check if a game has completed and fetch results if so.
        
        Args:
            game: The tracked game to check
            
        Returns:
            True if game was completed, False if still active
        """
        # Get player info
        player = await self.database.get_player_by_id(game.player_id)
        if not player:
            logger.error(f"Player {game.player_id} not found for game {game.game_id}")
            return False
        
        # Check if player is still in this game
        try:
            current_game = await self.riot_api.get_active_game_info(
                player.game_name,
                player.tag_line
            )
            
            if current_game:
                current_game_id = str(current_game.to_dict().get('gameId', ''))
                if current_game_id == game.game_id:
                    # Still in the same game
                    logger.debug(f"Game {game.game_id} still active for {player.riot_id}")
                    return False
        except PlayerNotInGameError:
            # Player not in game, so game must have ended
            pass
        
        # Game has ended - fetch match results
        logger.info(f"Game {game.game_id} has ended for {player.riot_id}, fetching results...")
        
        # Determine queue type for display/events (optional)
        queue_type = None
        if game.queue_type:
            # Find queue type by matching the value
            for qt in QueueType:
                if qt.value == game.queue_type:
                    queue_type = qt
                    break
        
        try:
            # Fetch match details using game_type from database
            match_info = await self.riot_api.get_match_for_game(
                game.game_id,
                game.game_type,
                region="na1"
            )
            
            if not match_info:
                logger.warning(f"No match data returned for game {game.game_id}, will retry")
                await self.database.update_game_error(
                    game.player_id,
                    game.game_id,
                    "No match data returned from API"
                )
                return False
            
            # Process match results
            game_result_data = None  # Dict for database storage
            game_result_obj = None   # Domain object for event
            duration_seconds = None
            
            # Extract result based on game type
            if game.game_type == 'LOL':
                # LoL game
                if hasattr(match_info, 'get_participant_result_by_name'):
                    result = match_info.get_participant_result_by_name(
                        player.game_name,
                        player.tag_line
                    )
                    if result:
                        game_result_data = result
                        duration_seconds = match_info.game_duration
                        # Create domain object for event
                        game_result_obj = LoLGameResult(
                            won=result.get('won', False),
                            duration_seconds=duration_seconds,
                            champion_played=result.get('champion_name', '')
                        )
            elif game.game_type == 'TFT':
                # TFT game
                if hasattr(match_info, 'get_placement_by_name'):
                    placement = match_info.get_placement_by_name(player.game_name, player.tag_line)
                    if placement is not None:
                        game_result_data = {'placement': placement}
                        duration_seconds = int(match_info.game_length) if hasattr(match_info, 'game_length') else None
                        # Create domain object for event
                        game_result_obj = TFTGameResult(
                            placement=placement,
                            duration_seconds=duration_seconds
                        )
            
            # Critical: Only complete game if we have results
            if not game_result_data:
                logger.warning(
                    f"No game result available for {game.game_id} - keeping game active for retry"
                )
                return False  # Don't complete the game without results
            
            # Create a TrackedGame domain object with results for event creation
            # (The 'game' from database is a SQLAlchemy model, not domain entity)
            tracked_game_entity = TrackedGame(
                player_id=game.player_id,
                game_id=game.game_id,
                game_type=game.game_type,
                status='COMPLETED',
                detected_at=game.detected_at,
                started_at=game.started_at,
                completed_at=datetime.utcnow(),
                queue_type=queue_type,
                game_result=game_result_obj,
                duration_seconds=duration_seconds,
                id=game.id
            )
            
            # Update game as completed in database
            await self.database.complete_tracked_game(
                game_id=game.id,  # Use the database ID
                game_result_data=game_result_data,
                duration_seconds=duration_seconds
            )
            
            logger.info(
                f"Completed game {game.game_id} for {player.riot_id} "
                f"(Result: {game_result_data})"
            )
            
            # Emit game completed event using proper event object
            try:
                event = self._create_game_end_event(player, tracked_game_entity)
                if event:
                    await self.event_publisher.publish_game_state_changed(event)
                    logger.debug(f"Published {event.get_event_type()} event for game {game.game_id}")
                else:
                    logger.error(f"Failed to create game end event for {game.game_id} - no results available")
            except Exception as e:
                logger.error(f"Failed to publish game completed event: {e}")
            
            return True
            
        except Exception as e:
            logger.error(
                f"Failed to fetch results for game {game.game_id}: {e}. "
                f"Will retry in next cycle"
            )
            await self.database.update_game_error(
                game.player_id,
                game.game_id,
                str(e)
            )
            return False