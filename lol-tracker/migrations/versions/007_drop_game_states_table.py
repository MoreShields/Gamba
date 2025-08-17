"""Drop legacy game_states table

Revision ID: 007
Revises: 006
Create Date: 2025-01-17

This migration removes the game_states table which was used by the legacy
state-transition polling model. We now use the game-centric model with
the tracked_games table exclusively.
"""
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '007'
down_revision = '006'
branch_labels = None
depends_on = None


def upgrade():
    # Drop the game_states table
    op.drop_table('game_states')


def downgrade():
    # Recreate the game_states table if needed
    op.create_table('game_states',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('player_id', sa.Integer(), nullable=False),
        sa.Column('status', sa.String(length=50), nullable=False),
        sa.Column('game_id', sa.String(length=20), nullable=True),
        sa.Column('queue_type', sa.String(length=50), nullable=True),
        sa.Column('game_result_data', sa.dialects.postgresql.JSONB(), nullable=True),
        sa.Column('duration_seconds', sa.Integer(), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('game_start_time', sa.DateTime(), nullable=True),
        sa.Column('game_end_time', sa.DateTime(), nullable=True),
        sa.Column('raw_api_response', sa.Text(), nullable=True),
        sa.ForeignKeyConstraint(['player_id'], ['tracked_players.id'], ondelete='CASCADE'),
        sa.PrimaryKeyConstraint('id')
    )
    op.create_index('idx_game_states_player_created', 'game_states', ['player_id', 'created_at'])
    op.create_index('idx_game_states_game_id', 'game_states', ['game_id'])
    op.create_index('idx_game_states_status', 'game_states', ['status'])
    op.create_index('idx_game_states_player_status', 'game_states', ['player_id', 'status'])