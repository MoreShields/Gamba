"""PostgreSQL integration coverage for tracked-games cleanup migration."""

import os
import subprocess
import sys
from pathlib import Path
from urllib.parse import urlparse

from sqlalchemy import create_engine, text

PROJECT_ROOT = Path(__file__).parent.parent


def _run_alembic(database_url: str, command: str, revision: str) -> None:
    parsed_url = urlparse(database_url)
    env = os.environ.copy()
    env.update(
        {
            "DATABASE_URL": database_url,
            "DATABASE_NAME": parsed_url.path.lstrip("/"),
            "RIOT_API_KEY": "migration-test",
            "TFT_RIOT_API_KEY": "migration-test",
            "ENVIRONMENT": "CI",
        }
    )
    result = subprocess.run(
        [sys.executable, "-m", "alembic", command, revision],
        cwd=PROJECT_ROOT,
        env=env,
        capture_output=True,
        text=True,
        timeout=30,
    )
    assert result.returncode == 0, result.stderr


def test_revision_008_normalizes_supported_queues_and_deletes_the_rest(
    postgres_container,
):
    database_url = postgres_container.get_connection_url()
    _run_alembic(database_url, "upgrade", "007")

    engine = create_engine(database_url)
    with engine.begin() as connection:
        connection.execute(text("DELETE FROM tracked_games"))
        connection.execute(text("DELETE FROM tracked_players"))

    # Other integration tests share this PostgreSQL container and may already
    # have applied 008. Its changes are data-only, so stamping 007 gives this
    # test an isolated chance to execute the cleanup against the seeded rows.
    _run_alembic(database_url, "stamp", "007")

    rows = [
        (1, "420", "ACTIVE", None, "{'gameQueueConfigId': 420}"),
        (2, "440", "COMPLETED", "TFT_UNKNOWN", '{"gameQueueConfigId": 440}'),
        (3, "1100", "COMPLETED", "QUEUE_1100", "{'gameQueueConfigId': 1100}"),
        (4, "1160", "ACTIVE", "TFT_UNKNOWN", "{'gameQueueConfigId': 1160}"),
        (5, "2400", "ACTIVE", "QUEUE_2400", "{'gameQueueConfigId': 2400}"),
        (6, "4310", "COMPLETED", None, '{"gameQueueConfigId": 4310}'),
        (7, "2450", "COMPLETED", "QUEUE_2450", "{'gameQueueConfigId': 2450}"),
        (8, "unknown", "ACTIVE", None, "not parseable"),
        (9, "supported", "COMPLETED", "TFT_RANKED", None),
        (10, "legacy-420", "COMPLETED", None, "{'queue_id': 420}"),
        (11, "legacy-1160", "ACTIVE", "TFT_UNKNOWN", '{"queue_id": 1160}'),
    ]
    with engine.begin() as connection:
        connection.execute(text("""
                INSERT INTO tracked_players
                    (id, game_name, tag_line, created_at, updated_at)
                VALUES
                    (1, 'MigrationPlayer', 'NA1', NOW(), NOW())
                """))
        connection.execute(
            text("""
                INSERT INTO tracked_games
                    (id, player_id, game_id, status, game_type, queue_type,
                     raw_api_response)
                VALUES
                    (:id, 1, :game_id, :status, 'LOL', :queue_type,
                     :raw_api_response)
                """),
            [
                {
                    "id": row_id,
                    "game_id": game_id,
                    "status": status,
                    "queue_type": queue_type,
                    "raw_api_response": raw_response,
                }
                for row_id, game_id, status, queue_type, raw_response in rows
            ],
        )

    _run_alembic(database_url, "upgrade", "008")

    with engine.connect() as connection:
        remaining = connection.execute(
            text("SELECT id, queue_type FROM tracked_games ORDER BY id")
        ).all()
    engine.dispose()

    assert remaining == [
        (1, "RANKED_SOLO_5x5"),
        (2, "RANKED_FLEX_SR"),
        (3, "TFT_RANKED"),
        (4, "TFT_RANKED_DOUBLE_UP"),
        (9, "TFT_RANKED"),
        (10, "RANKED_SOLO_5x5"),
        (11, "TFT_RANKED_DOUBLE_UP"),
    ]
