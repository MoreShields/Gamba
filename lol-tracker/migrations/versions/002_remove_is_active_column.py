"""Remove is_active column from tracked_players table

Revision ID: 002
Revises: 001
Create Date: 2025-08-05 12:00:00.000000

"""
from alembic import op
import sqlalchemy as sa

# revision identifiers, used by Alembic.
revision = '002'
down_revision = '001'
branch_labels = None
depends_on = None


def upgrade() -> None:
    # Drop the index on is_active column first
    op.drop_index('idx_tracked_players_active', table_name='tracked_players')
    
    # Remove the is_active column
    op.drop_column('tracked_players', 'is_active')


def downgrade() -> None:
    # Add back the is_active column with default value True
    op.add_column('tracked_players', sa.Column('is_active', sa.Boolean(), nullable=False, server_default=sa.text('true')))
    
    # Recreate the index
    op.create_index('idx_tracked_players_active', 'tracked_players', ['is_active'])