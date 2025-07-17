"""Initial schema for tracked players and game states

Revision ID: 001
Revises: 
Create Date: 2025-07-15 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision = '001'
down_revision = None
branch_labels = None
depends_on = None


def upgrade() -> None:
    # Create tracked_players table
    op.create_table('tracked_players',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('game_name', sa.String(length=16), nullable=False),
        sa.Column('tag_line', sa.String(length=5), nullable=False),
        sa.Column('puuid', sa.String(length=78), nullable=False),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('updated_at', sa.DateTime(), nullable=False),
        sa.Column('is_active', sa.Boolean(), nullable=False),
        sa.PrimaryKeyConstraint('id')
    )
    
    # Create indexes for tracked_players
    op.create_index('idx_tracked_players_game_name', 'tracked_players', ['game_name'])
    op.create_index('idx_tracked_players_puuid', 'tracked_players', ['puuid'])
    op.create_index('idx_tracked_players_active', 'tracked_players', ['is_active'])
    
    # Create game_states table
    op.create_table('game_states',
        sa.Column('id', sa.Integer(), autoincrement=True, nullable=False),
        sa.Column('player_id', sa.Integer(), nullable=False),
        sa.Column('status', sa.String(length=50), nullable=False),
        sa.Column('game_id', sa.String(length=20), nullable=True),
        sa.Column('queue_type', sa.String(length=50), nullable=True),
        sa.Column('won', sa.Boolean(), nullable=True),
        sa.Column('duration_seconds', sa.Integer(), nullable=True),
        sa.Column('champion_played', sa.String(length=50), nullable=True),
        sa.Column('created_at', sa.DateTime(), nullable=False),
        sa.Column('game_start_time', sa.DateTime(), nullable=True),
        sa.Column('game_end_time', sa.DateTime(), nullable=True),
        sa.Column('raw_api_response', sa.Text(), nullable=True),
        sa.ForeignKeyConstraint(['player_id'], ['tracked_players.id'], ),
        sa.PrimaryKeyConstraint('id')
    )
    
    # Create indexes for game_states
    op.create_index('idx_game_states_player_created', 'game_states', ['player_id', 'created_at'])
    op.create_index('idx_game_states_game_id', 'game_states', ['game_id'])
    op.create_index('idx_game_states_status', 'game_states', ['status'])
    op.create_index('idx_game_states_player_status', 'game_states', ['player_id', 'status'])


def downgrade() -> None:
    # Drop game_states table and its indexes
    op.drop_index('idx_game_states_player_status', table_name='game_states')
    op.drop_index('idx_game_states_status', table_name='game_states')
    op.drop_index('idx_game_states_game_id', table_name='game_states')
    op.drop_index('idx_game_states_player_created', table_name='game_states')
    op.drop_table('game_states')
    
    # Drop tracked_players table and its indexes
    op.drop_index('idx_tracked_players_active', table_name='tracked_players')
    op.drop_index('idx_tracked_players_puuid', table_name='tracked_players')
    op.drop_index('idx_tracked_players_game_name', table_name='tracked_players')
    op.drop_table('tracked_players')