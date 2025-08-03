# Base image with all build dependencies
FROM ubuntu:22.04

# Install system dependencies
RUN apt-get update && apt-get install -y \
    build-essential \
    curl \
    git \
    protobuf-compiler \
    python3.11 \
    python3.11-dev \
    python3.11-venv \
    python3-pip \
    pkg-config \
    libssl-dev \
    libffi-dev \
    && rm -rf /var/lib/apt/lists/*

# Install Go
ENV GO_VERSION=1.24.4
RUN curl -L https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz | tar -xz -C /usr/local
ENV PATH="/usr/local/go/bin:${PATH}"
ENV GOPATH="/go"
ENV PATH="${GOPATH}/bin:${PATH}"

# Install Go protobuf tools
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.31.0 && \
    go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.3.0

# Set up working directory to match GitHub Actions
WORKDIR /__w/Gamba/Gamba

# Create symlinks for Python
RUN ln -sf /usr/bin/python3.11 /usr/bin/python3 && \
    ln -sf /usr/bin/python3.11 /usr/bin/python

# Verify installations
RUN protoc --version && \
    go version && \
    python3.11 --version && \
    protoc-gen-go --version && \
    protoc-gen-go-grpc --version