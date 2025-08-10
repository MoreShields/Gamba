"""Constants for OpenTelemetry metrics."""

# Service name
SERVICE_NAME = "lol-tracker"

# Metric name prefixes
METRIC_PREFIX = "lol_tracker"

# Riot API metrics
RIOT_API_CALLS_TOTAL = f"{METRIC_PREFIX}.riot_api.calls_total"
RIOT_API_CALL_DURATION = f"{METRIC_PREFIX}.riot_api.call_duration"
RIOT_API_RATE_LIMITS = f"{METRIC_PREFIX}.riot_api.rate_limits_total"

# Game state metrics
GAME_STATE_CHANGES = f"{METRIC_PREFIX}.game_state.changes_total"
GAMES_DETECTED = f"{METRIC_PREFIX}.games.detected_total"
GAMES_COMPLETED = f"{METRIC_PREFIX}.games.completed_total"

# Message bus metrics
MESSAGES_PUBLISHED = f"{METRIC_PREFIX}.messages.published_total"
MESSAGE_PUBLISH_FAILURES = f"{METRIC_PREFIX}.messages.publish_failures_total"

# Polling loop metrics
POLLING_ITERATIONS = f"{METRIC_PREFIX}.polling.iterations_total"
POLLING_ERRORS = f"{METRIC_PREFIX}.polling.errors_total"

# Common label keys
LABEL_ENDPOINT_TYPE = "endpoint_type"
LABEL_API_KEY_TYPE = "api_key_type"
LABEL_STATUS_CODE = "status_code"
LABEL_ERROR_TYPE = "error_type"
LABEL_GAME_TYPE = "game_type"
LABEL_QUEUE_TYPE = "queue_type"
LABEL_EVENT_TYPE = "event_type"
LABEL_MESSAGE_SUBJECT = "subject"
LABEL_MESSAGE_STREAM = "stream"
LABEL_LOOP_TYPE = "loop_type"
LABEL_TRANSITION_TYPE = "transition_type"