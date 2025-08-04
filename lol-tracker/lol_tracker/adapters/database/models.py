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
)
from sqlalchemy.dialects.postgresql import UUID
from sqlalchemy.orm import declarative_base, Mapped, mapped_column, relationship
from sqlalchemy.sql import func
import uuid

Base = declarative_base()


class TrackedPlayer(Base):
    """Model for tracked League of Legends players."""

    __tablename__ = "tracked_players"

    id: Mapped[int] = mapped_column(Integer, primary_key=True, autoincrement=True)
    game_name: Mapped[str] = mapped_column(String(16), nullable=False)
    tag_line: Mapped[str] = mapped_column(String(5), nullable=False)  # Tag line without # (e.g., "gamba")
    puuid: Mapped[str] = mapped_column(String(78), nullable=False)  # Riot's persistent unique identifier

    # Tracking metadata
    created_at: Mapped[datetime] = mapped_column(DateTime, nullable=False, default=func.now())
    updated_at: Mapped[datetime] = mapped_column(
        DateTime, nullable=False, default=func.now(), onupdate=func.now()
    )

    # Relationships
    game_states: Mapped[List["GameState"]] = relationship(
        "GameState", back_populates="player", cascade="all, delete-orphan"
    )

    # Indexes for efficient querying
    __table_args__ = (
        Index("idx_tracked_players_game_name", "game_name"),
        Index("idx_tracked_players_puuid", "puuid"),
    )

    def __repr__(self) -> str:
        return f"<TrackedPlayer(game_name='{self.game_name}', puuid='{self.puuid}')>"


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

    # Game result information (populated when game ends)
    won: Mapped[Optional[bool]] = mapped_column(Boolean, nullable=True)
    duration_seconds: Mapped[Optional[int]] = mapped_column(Integer, nullable=True)
    champion_played: Mapped[Optional[str]] = mapped_column(String(50), nullable=True)

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
