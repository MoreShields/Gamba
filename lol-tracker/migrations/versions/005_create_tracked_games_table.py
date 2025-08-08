"""Create tracked_games table for game-centric architecture

Revision ID: 005
Revises: 004_remove_puuid_column
Create Date: 2024-01-08

This migration creates a new tracked_games table that represents each game
as a single row that progresses through its lifecycle, replacing the 
state-transition model in game_states.
"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects import postgresql

# revision identifiers
revision = '005'
down_revision = '004'
branch_labels = None
depends_on = None

def upgrade():
    # Create the new tracked_games table
    op.create_table('tracked_games',
        sa.Column('id', sa.Integer(), nullable=False),
        sa.Column('player_id', sa.Integer(), nullable=False),
        sa.Column('game_id', sa.String(20), nullable=False),
        sa.Column('status', sa.String(20), nullable=False, server_default='ACTIVE'),
        sa.Column('detected_at', sa.TIMESTAMP(), nullable=False, server_default=sa.func.now()),
        sa.Column('started_at', sa.TIMESTAMP(), nullable=True),
        sa.Column('completed_at', sa.TIMESTAMP(), nullable=True),
        sa.Column('queue_type', sa.String(50), nullable=True),
        sa.Column('game_result_data', postgresql.JSONB(), nullable=True),
        sa.Column('duration_seconds', sa.Integer(), nullable=True),
        sa.Column('raw_api_response', sa.Text(), nullable=True),
        sa.Column('last_error', sa.Text(), nullable=True),
        sa.PrimaryKeyConstraint('id'),
        sa.ForeignKeyConstraint(['player_id'], ['tracked_players.id'], ondelete='CASCADE'),
        sa.UniqueConstraint('player_id', 'game_id', name='uq_tracked_games_player_game')
    )
    
    # Create indexes for query performance
    op.create_index('idx_tracked_games_status', 'tracked_games', ['status'])
    op.create_index('idx_tracked_games_player_status', 'tracked_games', ['player_id', 'status'])
    op.create_index('idx_tracked_games_detected', 'tracked_games', [sa.text('detected_at DESC')])
    op.create_index('idx_tracked_games_completed', 'tracked_games', [sa.text('completed_at DESC')], 
                    postgresql_where=sa.text('completed_at IS NOT NULL'))
    
    # Add comment for documentation
    op.execute("COMMENT ON TABLE tracked_games IS 'Game-centric tracking where each row represents a complete game lifecycle'")

def downgrade():
    op.drop_index('idx_tracked_games_completed', table_name='tracked_games')
    op.drop_index('idx_tracked_games_detected', table_name='tracked_games')
    op.drop_index('idx_tracked_games_player_status', table_name='tracked_games')
    op.drop_index('idx_tracked_games_status', table_name='tracked_games')
    op.drop_table('tracked_games')