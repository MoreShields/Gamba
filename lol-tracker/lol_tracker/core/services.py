"""Core services for the lol-tracker service."""

from typing import Optional, Dict, Any
from datetime import datetime

from .entities import Player, GameState
from .enums import GameStatus, QueueType


class GameStateTransitionService:
    """Simplified service for handling game state transitions.
    
    Consolidates the complex domain service logic into a more pragmatic
    approach focused on the core business requirements.
    """
    
    def handle_riot_api_response(
        self,
        player: Player,
        current_state: GameState,
        api_response: Optional[Dict[str, Any]]
    ) -> tuple[GameState, bool]:
        """Process Riot API response and determine new game state.
        
        Args:
            player: The player being tracked
            current_state: Current game state
            api_response: Raw response from Riot API (None if not in game)
            
        Returns:
            Tuple of (new_game_state, state_changed)
            
        Raises:
            ValueError: If player cannot be tracked or API response is invalid
        """
        if not player.can_be_tracked():
            raise ValueError("Player cannot be tracked")
        
        # Player is not in game
        if api_response is None:
            return self._handle_not_in_game(current_state)
        
        # Player has active game data
        return self._handle_in_game(current_state, api_response)
    
    def _handle_not_in_game(self, current_state: GameState) -> tuple[GameState, bool]:
        """Handle when player is not in game."""
        if current_state.status == GameStatus.NOT_IN_GAME:
            return current_state, False
        
        # Player left/finished the game
        new_state = GameState(
            status=GameStatus.NOT_IN_GAME,
            player_id=current_state.player_id,
            game_id=current_state.game_id,  # Keep game_id for result updates
            queue_type=current_state.queue_type,
            game_result=current_state.game_result,  # Keep any existing game result
            game_start_time=current_state.game_start_time,
            game_end_time=datetime.utcnow()
        )
        
        return new_state, True
    
    def _handle_in_game(
        self,
        current_state: GameState,
        api_response: Dict[str, Any]
    ) -> tuple[GameState, bool]:
        """Handle when player is in game."""
        try:
            game_id = str(api_response["gameId"])
            queue_id = api_response["gameQueueConfigId"]
            game_start_time_ms = api_response.get("gameStartTime", 0)
            game_length_seconds = api_response.get("gameLength", 0)
            
            queue_type = QueueType.from_queue_id(queue_id)
            
            # Convert game start time from milliseconds to datetime
            game_start_time = None
            if game_start_time_ms > 0:
                game_start_time = datetime.fromtimestamp(game_start_time_ms / 1000)
            
            # Game length > 0 means the game has started
            is_in_game = game_length_seconds > 0
            
            if current_state.status == GameStatus.NOT_IN_GAME:
                if is_in_game:
                    # Transition to in-game
                    new_state = GameState.create_in_game(
                        player_id=current_state.player_id,
                        game_id=game_id,
                        queue_type=queue_type,
                        game_start_time=game_start_time
                    )
                    return new_state, True
                else:
                    # Still not in game (might be champion select)
                    return current_state, False
            
            elif current_state.status == GameStatus.IN_GAME:
                # Verify we're still in the same game
                if current_state.game_id == game_id:
                    return current_state, False
                else:
                    # Different game - create new in-game state
                    new_state = GameState.create_in_game(
                        player_id=current_state.player_id,
                        game_id=game_id,
                        queue_type=queue_type,
                        game_start_time=game_start_time
                    )
                    return new_state, True
            
            return current_state, False
            
        except KeyError as e:
            raise ValueError(f"Invalid API response format: missing key {e}")
        except Exception as e:
            raise ValueError(f"Error processing API response: {e}")