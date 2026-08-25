"""Remove tracked games from unsupported queues.

Revision ID: 008
Revises: 007
Create Date: 2026-08-25

Supported historical rows are normalized from the queue ID in the stored
Riot API response before unsupported rows are removed.
"""

import sqlalchemy as sa
from alembic import op

# revision identifiers, used by Alembic.
revision = "008"
down_revision = "007"
branch_labels = None
depends_on = None


def upgrade() -> None:
    connection = op.get_bind()

    # raw_api_response is commonly a Python dict repr (single quotes), but some
    # rows contain JSON (double quotes), so the expression accepts either form.
    connection.execute(sa.text("""
            WITH supported_queues(queue_id, queue_type) AS (
                VALUES
                    ('420', 'RANKED_SOLO_5x5'),
                    ('440', 'RANKED_FLEX_SR'),
                    ('1100', 'TFT_RANKED'),
                    ('1160', 'TFT_RANKED_DOUBLE_UP')
            )
            UPDATE tracked_games AS tracked_game
            SET queue_type = supported_queues.queue_type
            FROM supported_queues
            WHERE tracked_game.raw_api_response ~ (
                '[''"](gameQueueConfigId|queue_id)[''"]'
                || '[[:space:]]*:[[:space:]]*'
                || supported_queues.queue_id
                || '([^0-9]|$)'
            )
            """))

    connection.execute(sa.text("""
            DELETE FROM tracked_games
            WHERE queue_type IS NULL
               OR queue_type NOT IN (
                    'RANKED_SOLO_5x5',
                    'RANKED_FLEX_SR',
                    'TFT_RANKED',
                    'TFT_RANKED_DOUBLE_UP'
               )
            """))


def downgrade() -> None:
    """Do nothing because deleted tracked-game data is unrecoverable."""
