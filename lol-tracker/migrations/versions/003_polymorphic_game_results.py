"""Add polymorphic game result support with JSONB column

Revision ID: 003
Revises: 002
Create Date: 2025-08-06 12:00:00.000000

This migration transforms the game_states table to support multiple game types
(LoL and TFT) using a JSONB column for game-specific result data.
"""
from alembic import op
import sqlalchemy as sa
from sqlalchemy.dialects.postgresql import JSONB

# revision identifiers, used by Alembic.
revision = '003'
down_revision = '002'
branch_labels = None
depends_on = None


def upgrade() -> None:
    """
    Migration steps:
    1. Add new game_result_data JSONB column
    2. Migrate existing LoL data to JSON format
    3. Create indexes for performance
    4. Drop old LoL-specific columns
    """
    
    # Add new JSONB column for polymorphic game results
    op.add_column('game_states',
        sa.Column('game_result_data', JSONB, nullable=True)
    )
    
    # Migrate existing LoL game results to JSON format
    # This preserves all existing data in the new structure
    op.execute("""
        UPDATE game_states 
        SET game_result_data = jsonb_build_object(
            'won', won,
            'duration_seconds', duration_seconds,
            'champion_played', champion_played
        )
        WHERE won IS NOT NULL 
          AND duration_seconds IS NOT NULL 
          AND champion_played IS NOT NULL
    """)
    
    # Create indexes for better query performance
    op.create_index('idx_game_states_result_data', 'game_states', ['game_result_data'], 
                    postgresql_using='gin')
    
    # Create functional indexes for common queries
    # Index for LoL games (identified by queue type)
    op.execute("""
        CREATE INDEX idx_game_states_lol_won 
        ON game_states((game_result_data->>'won')) 
        WHERE queue_type NOT LIKE '%TFT%' OR queue_type IS NULL
    """)
    
    # Index for TFT games (identified by queue type containing 'TFT')
    op.execute("""
        CREATE INDEX idx_game_states_tft_placement
        ON game_states(((game_result_data->>'placement')::int))
        WHERE queue_type LIKE '%TFT%'
    """)
    
    # Verify data migration (in production, add more validation)
    from sqlalchemy import text
    result = op.get_bind().execute(text("""
        SELECT COUNT(*) as mismatched
        FROM game_states 
        WHERE won IS NOT NULL
          AND game_result_data IS NULL
    """)).scalar()
    
    if result and result > 0:
        raise Exception(f"Data migration failed: {result} records not migrated properly")
    
    # Drop old LoL-specific columns
    # Note: duration_seconds is kept as it's common to both LoL and TFT
    op.drop_column('game_states', 'won')
    op.drop_column('game_states', 'champion_played')
    # Keep duration_seconds for backward compatibility and because both games use it


def downgrade() -> None:
    """
    Rollback strategy:
    1. Re-add old columns
    2. Restore data from JSON for LoL games
    3. Drop new column and indexes
    """
    
    # Step 1: Re-add the old LoL-specific columns
    op.add_column('game_states',
        sa.Column('won', sa.Boolean(), nullable=True)
    )
    op.add_column('game_states',
        sa.Column('champion_played', sa.String(length=50), nullable=True)
    )
    
    # Step 2: Restore data from JSON back to columns for LoL games
    # Use queue_type to identify LoL games
    op.execute("""
        UPDATE game_states 
        SET 
            won = (game_result_data->>'won')::boolean,
            champion_played = game_result_data->>'champion_played'
        WHERE queue_type IN ('RANKED_SOLO_5x5', 'RANKED_FLEX_SR', 'ARAM', 'NORMAL_DRAFT', 'NORMAL_BLIND', 'CLASH', 'ARENA')
          AND game_result_data IS NOT NULL
    """)
    
    # Step 3: Drop the functional indexes
    op.execute("DROP INDEX IF EXISTS idx_game_states_lol_won")
    op.execute("DROP INDEX IF EXISTS idx_game_states_tft_placement")
    
    # Step 4: Drop other indexes
    op.drop_index('idx_game_states_result_data', table_name='game_states')
    
    # Step 5: Drop new column
    op.drop_column('game_states', 'game_result_data')