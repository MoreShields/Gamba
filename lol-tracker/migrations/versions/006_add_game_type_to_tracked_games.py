"""Add game_type column to tracked_games table

Revision ID: 006
Revises: 005
Create Date: 2025-01-17

"""
import ast
from alembic import op
import sqlalchemy as sa


# revision identifiers, used by Alembic.
revision = '006'
down_revision = '005'
branch_labels = None
depends_on = None


def upgrade():
    # Add nullable column first
    op.add_column('tracked_games', 
        sa.Column('game_type', sa.String(10), nullable=True))
    
    # Backfill by parsing raw_api_response
    connection = op.get_bind()
    result = connection.execute(
        sa.text("SELECT id, raw_api_response FROM tracked_games WHERE game_type IS NULL")
    )
    
    for row in result:
        game_type = 'LOL'  # Default
        if row.raw_api_response:
            try:
                # Parse the string representation of the dict
                raw_data = ast.literal_eval(row.raw_api_response)
                
                # Check for game_type marker
                if 'game_type' in raw_data:
                    game_type = raw_data['game_type']
                elif raw_data.get('gameMode') == 'TFT':
                    game_type = 'TFT'
                # Additional check for TFT queue IDs
                elif raw_data.get('gameQueueConfigId') in [1090, 1100, 1110, 1120, 1130, 1140, 1150, 1160]:
                    game_type = 'TFT'
            except (ValueError, SyntaxError, TypeError):
                # If we can't parse, default to LOL
                pass
        
        connection.execute(
            sa.text(f"UPDATE tracked_games SET game_type = :game_type WHERE id = :id"),
            {"game_type": game_type, "id": row.id}
        )
    
    # Make non-nullable after backfill
    op.alter_column('tracked_games', 'game_type', nullable=False)


def downgrade():
    op.drop_column('tracked_games', 'game_type')