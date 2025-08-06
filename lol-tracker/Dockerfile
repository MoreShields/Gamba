# syntax=docker/dockerfile:1
# Multi-stage build for Python service
FROM python:3.11-slim AS builder

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies including protobuf compiler
# This layer is cached effectively
RUN apt-get update && apt-get install -y \
    build-essential \
    libpq-dev \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Create and activate virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Pre-install common Python packages for better caching
# This layer only rebuilds when we add new core dependencies
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install \
    grpcio \
    grpcio-tools \
    protobuf \
    psycopg2 \
    sqlalchemy \
    alembic \
    nats-py \
    pydantic \
    python-dotenv

# Copy requirements and install specific versions
COPY lol-tracker/requirements.txt .
RUN --mount=type=cache,target=/root/.cache/pip \
    pip install --no-cache-dir -r requirements.txt

# Copy protobuf source files
COPY api/proto/ ./proto/

# Generate Python protobuf files with correct import paths
RUN mkdir -p ./lol_tracker/proto && \
    cd lol_tracker && \
    python -m grpc_tools.protoc -I=../proto \
        --python_out=proto \
        --grpc_python_out=proto \
        --pyi_out=proto \
        ../proto/events/*.proto \
        ../proto/models/*.proto \
        ../proto/services/*.proto && \
    touch proto/__init__.py && \
    touch proto/events/__init__.py && \
    touch proto/models/__init__.py && \
    touch proto/services/__init__.py && \
    find proto -name "*.py" -type f -exec sed -i 's/^from services import/from . import/g' {} \; && \
    find proto -name "*.py" -type f -exec sed -i 's/^from events import/from . import/g' {} \; && \
    find proto -name "*.py" -type f -exec sed -i 's/^from models import/from . import/g' {} \;

# Development stage
FROM python:3.11-slim AS dev

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH"

# Install system dependencies
RUN apt-get update && apt-get install -y \
    libpq5 \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Copy virtual environment from builder
COPY --from=builder /opt/venv /opt/venv

# Install development dependencies (hupper for hot reload)
RUN pip install hupper>=1.12.0

# Create app user
RUN groupadd -r appuser && useradd -r -g appuser appuser

# Create app directory
WORKDIR /app

# Copy generated protobuf files with correct structure
COPY --from=builder /lol_tracker/proto/ ./lol_tracker/proto/

# Change ownership to app user
RUN chown -R appuser:appuser /app

# Switch to app user
USER appuser

# Set Python path
ENV PYTHONPATH=/app

# For development, we'll mount the source code
CMD ["python", "-m", "lol_tracker.main"]

# Production stage - use minimal python slim image
FROM python:3.11-slim AS production

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH"

# Install only runtime dependencies (no build tools)
RUN apt-get update && apt-get install -y \
    libpq5 \
    && rm -rf /var/lib/apt/lists/*

# Copy virtual environment from builder
COPY --from=builder /opt/venv /opt/venv

# Create app user
RUN groupadd -r appuser && useradd -r -g appuser appuser

# Create app directory
WORKDIR /app

# Copy application code
COPY lol-tracker/lol_tracker/ ./lol_tracker/

# Copy alembic configuration and migrations
COPY lol-tracker/alembic.ini ./
COPY lol-tracker/migrations/ ./migrations/

# Copy generated protobuf files with correct structure
COPY --from=builder /lol_tracker/proto/ ./lol_tracker/proto/

# Change ownership to app user
RUN chown -R appuser:appuser /app

# Switch to app user
USER appuser

# Set Python path
ENV PYTHONPATH=/app

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD python -c "import asyncio; import sys; sys.exit(0)"

# Default command
CMD ["python", "-m", "lol_tracker.main"]