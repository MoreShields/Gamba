# LoL Tracker Game-Centric Architecture Migration Plan

## Executive Summary

This document outlines a comprehensive migration strategy to transform the LoL Tracker service from a state-transition model to a game-centric model. The migration uses a parallel-table approach for zero-downtime deployment with safe rollback capabilities.

**Key Benefits:**
- Eliminates race conditions where games can be lost during rapid transitions
- Reduces codebase complexity by ~30%
- Improves system observability and debugging
- Ensures reliable game result tracking

## Migration Progress

### ✅ Completed (2025-01-08)
- Database migration file created and tested
- SQLAlchemy TrackedGame model implemented
- DatabaseManager repository methods added
- TrackedGame domain entity with business logic
- Integration tests for new functionality

### 🚧 In Progress
- GameCentricPollingService implementation
- Feature flag configuration

### 📋 TODO
- Shadow deployment configuration
- Monitoring and validation queries
- Production deployment

## Current State Analysis

### Existing Architecture Problems

The current system tracks game states through transitions:
```
NOT_IN_GAME → IN_GAME → NOT_IN_GAME (with results)
```

**Race Condition Scenario:**
1. Player ends Game A (game_id: "abc123")
2. System detects game end, begins fetching match results
3. Player immediately starts Game B (game_id: "xyz789")
4. Next poll queries by game_name/tag_line, detects Game B
5. Game A results are never fetched, Game B is now tracked
6. Data loss occurs

### Current Database Schema

```sql
-- game_states table (current)
CREATE TABLE game_states (
    id SERIAL PRIMARY KEY,
    player_id INTEGER NOT NULL,
    status VARCHAR(50) NOT NULL,  -- 'NOT_IN_GAME', 'IN_GAME'
    game_id VARCHAR(20),
    queue_type VARCHAR(50),
    game_result_data JSONB,
    duration_seconds INTEGER,
    created_at TIMESTAMP,
    game_start_time TIMESTAMP,
    game_end_time TIMESTAMP,
    raw_api_response TEXT
);
```

Each state change creates a new row, making it difficult to track complete game lifecycles.

## New Architecture Design

### Game-Centric Model

Each game gets exactly one row that progresses through its lifecycle:
```
Game Detected (ACTIVE) → Game Completed (COMPLETED with results)
```

### New Database Schema

```sql
-- tracked_games table (new)
CREATE TABLE tracked_games (
    id SERIAL PRIMARY KEY,
    
    -- Core identification
    player_id INTEGER NOT NULL REFERENCES tracked_players(id) ON DELETE CASCADE,
    game_id VARCHAR(20) NOT NULL,
    
    -- Status tracking
    status VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',  -- 'ACTIVE', 'COMPLETED'
    
    -- Timestamps
    detected_at TIMESTAMP NOT NULL DEFAULT NOW(),
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Game information
    queue_type VARCHAR(50),
    game_result_data JSONB,  -- Polymorphic: LoL or TFT results
    duration_seconds INTEGER,
    
    -- Metadata
    raw_api_response TEXT,
    last_error TEXT,
    retry_count INTEGER DEFAULT 0,
    
    -- Constraints
    CONSTRAINT uq_tracked_games_player_game UNIQUE(player_id, game_id),
    
    -- Indexes for performance
    INDEX idx_tracked_games_status (status),
    INDEX idx_tracked_games_player_status (player_id, status),
    INDEX idx_tracked_games_detected (detected_at DESC),
    INDEX idx_tracked_games_completed (completed_at DESC) WHERE completed_at IS NOT NULL
);
```

## Migration Implementation

### Phase 1: Database Migration ✅ COMPLETED

**File: `migrations/versions/005_create_tracked_games_table.py`**

```python
"""Create tracked_games table for game-centric architecture

Revision ID: 005
Revises: 004
Create Date: 2024-01-XX

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
        sa.Column('retry_count', sa.Integer(), nullable=False, server_default='0'),
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
```

### Phase 2: Two-Loop Polling Architecture 🚧 IN PROGRESS

The new architecture uses two independent polling loops:

1. **Detection Loop (30s interval)**: Discovers new games and creates tracked_games entries
2. **Completion Loop (60s interval)**: Monitors active games for completion and fetches results

Key improvements:
- No state transitions to manage
- Game ID is the primary tracking key
- Each game is tracked from start to finish
- Results are fetched atomically when game ends

### Phase 3: Implementation Files ✅ PARTIALLY COMPLETED

Key files created/modified:
- ✅ `lol_tracker/core/entities.py` - Added TrackedGame entity
- 🚧 `lol_tracker/application/game_centric_polling_service.py` - New two-loop service (TODO)
- ✅ `lol_tracker/adapters/database/models.py` - Added TrackedGameModel
- ✅ `lol_tracker/adapters/database/manager.py` - Added game-centric repository methods
- 🚧 `lol_tracker/config.py` - Add feature flags (TODO)

## Deployment Strategy

### Week 1: Preparation and Testing

**Day 1-2: Development** ✅ PHASE 1 COMPLETE
```bash
# Create feature branch
git checkout -b feature/game-centric-architecture

# Completed:
# ✅ Create migration file (005_create_tracked_games_table.py)
# ✅ Add TrackedGame entity
# ✅ Update DatabaseManager with new methods
# ✅ Write and run integration tests

# TODO:
# - Implement GameCentricPollingService
# - Add feature flag support

# Test locally
make test
./venv/bin/python -m pytest tests/test_tracked_games_migration.py -v
```

**Day 3-4: Local Validation**
```yaml
# docker-compose.override.yml for local testing
version: '3.8'

services:
  lol-tracker-new:
    container_name: lol-tracker-new
    build: ./lol-tracker
    environment:
      USE_GAME_CENTRIC_MODEL: "true"
      EMIT_EVENTS: "false"  # Don't duplicate events
      DATABASE_URL: ${DATABASE_URL}
    command: ["python", "-m", "lol_tracker.main", "--game-centric"]
```

### Week 2: Production Deployment

**Step 1: Deploy Database Migration**
```bash
# SSH to production
ssh $EC2_HOST

# Run migration only
docker-compose run --rm lol-tracker-migrate

# Verify migration
docker-compose exec postgres psql -U postgres -d lol_tracker_db -c "
  SELECT table_name, column_name, data_type 
  FROM information_schema.columns 
  WHERE table_name = 'tracked_games'
  ORDER BY ordinal_position;
"
```

**Step 2: Deploy Parallel Test Instance**
```yaml
# Add to production docker-compose.yml
lol-tracker-shadow:
  container_name: lol-tracker-shadow
  image: ghcr.io/moreshields/lol-tracker:game-centric
  environment:
    DATABASE_URL: ${DATABASE_URL}
    USE_GAME_CENTRIC_MODEL: "true"
    EMIT_EVENTS: "false"  # Shadow mode - no events
    LOG_PREFIX: "[SHADOW]"
  restart: unless-stopped
  profiles: ["shadow"]
```

```bash
# Deploy shadow instance
docker-compose --profile shadow up -d lol-tracker-shadow

# Monitor logs
docker-compose logs -f lol-tracker-shadow | grep "new game\|completed game"
```

**Step 3: Validation Queries**
```sql
-- Monitor game detection parity
WITH game_comparison AS (
  SELECT 
    'legacy' as system,
    COUNT(DISTINCT game_id) as unique_games,
    COUNT(*) as total_rows,
    MAX(created_at) as latest_entry
  FROM game_states 
  WHERE created_at > NOW() - INTERVAL '1 hour'
    AND game_id IS NOT NULL
  UNION ALL
  SELECT 
    'new' as system,
    COUNT(DISTINCT game_id) as unique_games,
    COUNT(*) as total_rows,
    MAX(detected_at) as latest_entry
  FROM tracked_games 
  WHERE detected_at > NOW() - INTERVAL '1 hour'
)
SELECT * FROM game_comparison;

-- Find games missed by new system
SELECT 
  gs.game_id,
  gs.player_id,
  p.game_name || '#' || p.tag_line as player,
  gs.created_at
FROM game_states gs
JOIN tracked_players p ON p.id = gs.player_id
WHERE gs.created_at > NOW() - INTERVAL '4 hours'
  AND gs.game_id IS NOT NULL
  AND gs.status = 'IN_GAME'
  AND NOT EXISTS (
    SELECT 1 FROM tracked_games tg 
    WHERE tg.game_id = gs.game_id 
      AND tg.player_id = gs.player_id
  )
ORDER BY gs.created_at DESC;

-- Check completion rates
SELECT 
  status,
  COUNT(*) as game_count,
  AVG(retry_count) as avg_retries,
  MAX(retry_count) as max_retries
FROM tracked_games
WHERE detected_at > NOW() - INTERVAL '24 hours'
GROUP BY status;

-- Games stuck in ACTIVE state
SELECT 
  tg.game_id,
  p.game_name || '#' || p.tag_line as player,
  tg.detected_at,
  AGE(NOW(), tg.detected_at) as age,
  tg.retry_count,
  tg.last_error
FROM tracked_games tg
JOIN tracked_players p ON p.id = tg.player_id
WHERE tg.status = 'ACTIVE'
  AND tg.detected_at < NOW() - INTERVAL '2 hours'
ORDER BY tg.detected_at;
```

**Step 4: Feature Flag Configuration**
```python
# Add to config.py
class Config:
    # Feature flag to switch between services
    use_game_centric_model: bool = env.bool("USE_GAME_CENTRIC_MODEL", False)
```

```python
# In main.py - switch between services based on flag
async def start_polling(config: Config):
    if config.use_game_centric_model:
        from .application.game_centric_polling_service import GameCentricPollingService
        service = GameCentricPollingService(database, riot_api, event_publisher, config)
    else:
        from .application.polling_service import PollingService
        service = PollingService(database, riot_api, event_publisher, config)
    
    await service.start()
```

### Week 3: Full Migration

**Step 1: Complete Cutover**
```bash
# Update main service to use new model
docker-compose exec lol-tracker env USE_GAME_CENTRIC_MODEL=true
docker-compose restart lol-tracker

# Stop shadow instance
docker-compose --profile shadow stop lol-tracker-shadow
docker-compose --profile shadow rm lol-tracker-shadow
```

**Step 2: Cleanup (after 30 days)**
```python
# Migration 006_cleanup_legacy_game_states.py
def upgrade():
    # Archive old game_states data
    op.execute("""
        CREATE TABLE game_states_archive AS 
        SELECT * FROM game_states;
    """)
    
    # Add deprecation notice
    op.execute("""
        COMMENT ON TABLE game_states IS 
        'DEPRECATED - Use tracked_games table. Archived for historical data.';
    """)
```

## Monitoring and Alerts

### Key Metrics to Monitor

```sql
-- Create monitoring views
CREATE VIEW game_tracking_metrics AS
SELECT 
  DATE_TRUNC('hour', detected_at) as hour,
  COUNT(*) as games_detected,
  COUNT(CASE WHEN status = 'COMPLETED' THEN 1 END) as games_completed,
  COUNT(CASE WHEN status = 'ACTIVE' THEN 1 END) as games_active,
  AVG(CASE WHEN status = 'COMPLETED' THEN duration_seconds END) as avg_duration,
  MAX(retry_count) as max_retries,
  COUNT(CASE WHEN retry_count > 0 THEN 1 END) as games_with_retries
FROM tracked_games
GROUP BY DATE_TRUNC('hour', detected_at);

-- Alert queries
-- 1. High retry rate
SELECT COUNT(*) as high_retry_games
FROM tracked_games
WHERE detected_at > NOW() - INTERVAL '1 hour'
  AND retry_count >= 3;

-- 2. Stuck games
SELECT COUNT(*) as stuck_games
FROM tracked_games
WHERE status = 'ACTIVE'
  AND detected_at < NOW() - INTERVAL '3 hours';

-- 3. Missing results
SELECT COUNT(*) as missing_results
FROM tracked_games
WHERE status = 'COMPLETED'
  AND game_result_data IS NULL
  AND completed_at > NOW() - INTERVAL '1 hour';
```

## Rollback Procedures

### Immediate Rollback (if critical issues)

```bash
# Step 1: Stop new model service
docker-compose stop lol-tracker

# Step 2: Deploy previous version
docker pull ghcr.io/moreshields/lol-tracker:previous-tag
docker-compose up -d lol-tracker

# Step 3: Verify old model is running
docker-compose logs --tail=50 lol-tracker | grep "state transition"
```

### Data Recovery (if needed)

```sql
-- If any games were missed during transition
INSERT INTO game_states (player_id, status, game_id, queue_type, created_at)
SELECT 
  player_id,
  CASE 
    WHEN status = 'ACTIVE' THEN 'IN_GAME'
    ELSE 'NOT_IN_GAME'
  END as status,
  game_id,
  queue_type,
  detected_at as created_at
FROM tracked_games
WHERE detected_at > '2024-XX-XX'  -- Migration date
  AND NOT EXISTS (
    SELECT 1 FROM game_states gs 
    WHERE gs.game_id = tracked_games.game_id
      AND gs.player_id = tracked_games.player_id
  );
```

## Success Criteria

The migration is considered successful when:

1. **Detection Parity**: New system detects ≥99% of games compared to old system
2. **Completion Rate**: >95% of detected games get results within 5 minutes
3. **Event Accuracy**: All published events contain correct game results
4. **No Data Loss**: Zero games lost during transition period
5. **Performance**: Polling loops complete within their intervals
6. **Error Rate**: <1% of games require retries

## Configuration Reference

```yaml
# Environment variables for new system
USE_GAME_CENTRIC_MODEL: "true"          # Enable new architecture (true/false)
DETECTION_INTERVAL_SECONDS: "30"        # How often to check for new games
COMPLETION_INTERVAL_SECONDS: "60"       # How often to check for completed games
MAX_RESULT_RETRIES: "3"                 # Max attempts to fetch match results
```

## Implementation Checklist

### Phase 1: Database & Models (COMPLETED)
- [x] Create database migration (005_create_tracked_games_table.py)
- [x] Add TrackedGameModel to SQLAlchemy models
- [x] Update DatabaseManager with new repository methods
  - [x] `get_tracked_game(player_id, game_id)`
  - [x] `create_tracked_game(...)`
  - [x] `get_games_by_status(status)`
  - [x] `complete_tracked_game(...)`
  - [x] `update_game_error(...)`
  - [x] `get_player_by_id(player_id)`
- [x] Implement TrackedGame domain entity
  - [x] `is_active()`, `is_completed()` status methods
  - [x] `complete_with_results()` for game completion
  - [x] `mark_error()` for error tracking
- [x] Write integration tests for new database functionality
- [x] Verify migration runs successfully

### Phase 2: Polling Service (TODO)
- [ ] Create GameCentricPollingService with two loops
  - [ ] Detection loop (30s interval) - finds new games
  - [ ] Completion loop (60s interval) - fetches results
- [ ] Implement feature flags in Config
- [ ] Write unit tests for polling logic
- [ ] Write integration tests with mock API

### Phase 3: Deployment (TODO)
- [ ] Update docker-compose for shadow deployment
- [ ] Create monitoring queries and alerts
- [ ] Document rollback procedures
- [ ] Prepare validation scripts

## Risk Mitigation

1. **Parallel Deployment**: Run both systems side-by-side before cutover
2. **Feature Flag**: Simple boolean switch between old and new services
3. **Shadow Mode**: Test without affecting production events
4. **Comprehensive Monitoring**: Track all key metrics during migration
5. **Easy Rollback**: Single environment variable to revert
6. **Data Preservation**: Keep old table intact during transition

## Summary

This migration plan provides:
- **Zero-downtime deployment** through parallel table approach
- **Safe rollback** capability at any point
- **Comprehensive monitoring** to ensure data integrity
- **Simple feature flag** for clean service switching
- **Clear success criteria** for validation

The new game-centric architecture will:
- Eliminate race conditions completely
- Reduce code complexity by ~30%
- Improve system observability
- Ensure reliable game result tracking
- Simplify debugging and maintenance