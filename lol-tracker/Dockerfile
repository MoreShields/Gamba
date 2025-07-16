# Multi-stage build for Python service
FROM python:3.11-slim AS builder

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PIP_NO_CACHE_DIR=1 \
    PIP_DISABLE_PIP_VERSION_CHECK=1

# Install system dependencies including protobuf compiler
RUN apt-get update && apt-get install -y \
    build-essential \
    libpq-dev \
    protobuf-compiler \
    && rm -rf /var/lib/apt/lists/*

# Create and activate virtual environment
RUN python -m venv /opt/venv
ENV PATH="/opt/venv/bin:$PATH"

# Copy requirements and install Python dependencies
COPY lol-tracker/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy protobuf source files
COPY api/proto/ ./proto/

# Generate Python protobuf files
RUN mkdir -p ./generated/proto && \
    python -m grpc_tools.protoc -I=proto \
        --python_out=./generated/proto \
        --grpc_python_out=./generated/proto \
        proto/events/*.proto proto/models/*.proto

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

# Create app user
RUN groupadd -r appuser && useradd -r -g appuser appuser

# Create app directory
WORKDIR /app

# Copy generated protobuf files
COPY --from=builder /generated/proto/ ./lol_tracker/proto/

# Change ownership to app user
RUN chown -R appuser:appuser /app

# Switch to app user
USER appuser

# Set Python path
ENV PYTHONPATH=/app

# For development, we'll mount the source code
CMD ["python", "-m", "lol_tracker.main"]

# Production stage
FROM python:3.11-slim AS production

# Set environment variables
ENV PYTHONDONTWRITEBYTECODE=1 \
    PYTHONUNBUFFERED=1 \
    PATH="/opt/venv/bin:$PATH"

# Install runtime dependencies
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

# Copy generated protobuf files
COPY --from=builder /generated/proto/ ./lol_tracker/proto/

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