"""SQLAlchemy models for LoL Tracker service."""

from datetime import datetime
from typing import Optional, List

from sqlalchemy import (
    String,
    DateTime,
    Integer,
    Boolean,
    ForeignKey,
    Text,
    Index,
    UniqueConstraint,
)
from sqlalchemy.dialects.postgresql import JSONB
from sqlalchemy.orm import declarative_base, Mapped, mapped_column, relationship
from sqlalchemy.sql import func

Base = declarative_base()


class TrackedPlayer(Base):
    """Model for tracked League of Legends players."""

    __tablename__ = "tracked_players"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    game_name: Mapped[str] = mapped_column(String(16), nullable=False)
    tag_line: Mapped[str] = mapped_column(String(5), nullable=False)  # Tag line without # (e.g., "gamba")
    # Note: puuid column removed in migration 004 - PUUIDs are API-key specific

    # Tracking metadata
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=func.now())
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=func.now(), onupdate=func.now()
    )

    # Relationships
    game_states: Mapped[List["GameState"]] = relationship(
        "GameState", back_populates="player", cascade="all, delete-orphan"
    )
    tracked_games: Mapped[List["TrackedGame"]] = relationship(
        "TrackedGame", back_populates="player", cascade="all, delete-orphan"
    )

    # Indexes for efficient querying
    __table_args__ = (
        Index("idx_tracked_players_game_name", "game_name"),
        # Unique constraint to prevent duplicate tracking
        Index("uq_tracked_players_riot_id", "game_name", "tag_line", unique=True),
    )

    def __repr__(self) -> str:
        return f"<TrackedPlayer(game_name='{self.game_name}', tag_line='{self.tag_line}')>"


class GameState(Base):
    """Model for tracking game state changes of players."""

    __tablename__ = "game_states"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    player_id: Mapped[int] = mapped_column(
        Integer, ForeignKey("tracked_players.id", ondelete="CASCADE"), nullable=False
    )

    # Game state information
    status: Mapped[str] = mapped_column(
        String(50), nullable=False
    )  # NOT_IN_GAME, IN_CHAMPION_SELECT, IN_GAME
    game_id: Mapped[Optional[str]] = mapped_column(String(20), nullable=True)  # Riot's game ID
    queue_type: Mapped[Optional[str]] = mapped_column(
        String(50), nullable=True
    )  # Queue type (e.g., "RANKED_SOLO_5x5")

    # Polymorphic game result data stored as JSON
    game_result_data: Mapped[Optional[dict]] = mapped_column(JSONB, nullable=True)
    
    # Common fields for all game types
    duration_seconds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)

    # Timestamps
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=func.now())
    game_start_time: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    game_end_time: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)

    # Additional metadata
    raw_api_response: Mapped[Optional[str]] = mapped_column(
        Text, nullable=True
    )  # Store raw Riot API response for debugging

    # Relationships
    player: Mapped["TrackedPlayer"] = relationship("TrackedPlayer", back_populates="game_states")

    # Indexes for efficient querying
    __table_args__ = (
        Index("idx_game_states_player_created", "player_id", "created_at"),
        Index("idx_game_states_game_id", "game_id"),
        Index("idx_game_states_status", "status"),
        Index("idx_game_states_player_status", "player_id", "status"),
    )

    def __repr__(self) -> str:
        return f"<GameState(player_id={self.player_id}, status='{self.status}', game_id='{self.game_id}')>"


class TrackedGame(Base):
    """Represents a tracked game in the game-centric model.
    
    Each row represents a complete game lifecycle from detection to completion.
    """
    __tablename__ = "tracked_games"

    # Primary key
    id: Mapped[int] = mapped_column(Integer, primary_key=True)
    
    # Foreign key to player
    player_id: Mapped[int] = mapped_column(
        Integer, ForeignKey("tracked_players.id", ondelete="CASCADE"), nullable=False
    )
    
    # Core game identification
    game_id: Mapped[str] = mapped_column(String(20), nullable=False)
    status: Mapped[str] = mapped_column(String(20), nullable=False, default="ACTIVE")
    
    # Timestamps
    detected_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=func.now())
    started_at: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    completed_at: Mapped[Optional[datetime]] = mapped_column(DateTime, nullable=True)
    
    # Game information
    queue_type: Mapped[Optional[str]] = mapped_column(String(50), nullable=True)
    game_result_data: Mapped[Optional[dict]] = mapped_column(JSONB, nullable=True)
    duration_seconds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    
    # Metadata
    raw_api_response: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    last_error: Mapped[Optional[str]] = mapped_column(Text, nullable=True)
    
    # Relationships
    player: Mapped["TrackedPlayer"] = relationship("TrackedPlayer", back_populates="tracked_games")
    
    # Constraints and indexes
    __table_args__ = (
        UniqueConstraint("player_id", "game_id", name="uq_tracked_games_player_game"),
        Index("idx_tracked_games_status", "status"),
        Index("idx_tracked_games_player_status", "player_id", "status"),
        Index("idx_tracked_games_detected", "detected_at"),
        Index("idx_tracked_games_completed", "completed_at"),
    )
    
    def __repr__(self) -> str:
        return f"<TrackedGame(id={self.id}, player_id={self.player_id}, game_id='{self.game_id}', status='{self.status}')>"
