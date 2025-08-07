"""Remove PUUID column as it's API-key specific

Revision ID: 004
Revises: 003
Create Date: 2025-08-07 12:00:00.000000

PUUIDs are encrypted per API key. Since we now have separate API keys
for LoL and TFT, we need to fetch PUUIDs dynamically with the correct
key instead of storing them.

"""
from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision = '004'
down_revision = '003'
branch_labels = None
depends_on = None


def upgrade() -> None:
    # Drop the puuid index first
    op.drop_index('idx_tracked_players_puuid', table_name='tracked_players')
    
    # Drop the puuid column
    op.drop_column('tracked_players', 'puuid')
    
    # Add a unique constraint on (game_name, tag_line) to prevent duplicates
    op.create_unique_constraint(
        'uq_tracked_players_riot_id',
        'tracked_players',
        ['game_name', 'tag_line']
    )


def downgrade() -> None:
    # Remove the unique constraint
    op.drop_constraint('uq_tracked_players_riot_id', 'tracked_players', type_='unique')
    
    # Re-add the puuid column (nullable for safety)
    op.add_column('tracked_players', sa.Column('puuid', sa.String(length=78), nullable=True))
    
    # Re-create the index
    op.create_index('idx_tracked_players_puuid', 'tracked_players', ['puuid'])