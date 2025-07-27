# Development stage
FROM golang:1.24-alpine AS dev

# Install development dependencies
RUN apk add --no-cache git protobuf protobuf-dev

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Install protobuf Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

# Ensure Go bin directory is in PATH
ENV PATH="/go/bin:${PATH}"

WORKDIR /app

# Copy api proto files
COPY api/proto ../api/proto

# Copy go mod files
COPY discord-client/go.mod discord-client/go.sum* ./

# Download dependencies
RUN go mod download

# Generate protobuf code
RUN mkdir -p proto && \
    protoc -I=../api/proto \
        --go_out=proto --go_opt=paths=source_relative \
        --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
        ../api/proto/events/*.proto ../api/proto/models/*.proto ../api/proto/services/*.proto

# Copy discord-client source code
COPY discord-client ./

# Expose port if needed for debugging
EXPOSE 2345

CMD ["air"]

# Production builder stage
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git protobuf protobuf-dev

# Install protobuf Go plugins
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

WORKDIR /app

# Copy api proto files
COPY api/proto ../api/proto

# Copy go mod files
COPY discord-client/go.mod discord-client/go.sum* ./

# Download dependencies
RUN go mod download

# Generate protobuf code
RUN mkdir -p proto && \
    protoc -I=../api/proto \
        --go_out=proto --go_opt=paths=source_relative \
        --go-grpc_out=proto --go-grpc_opt=paths=source_relative \
        ../api/proto/events/*.proto ../api/proto/models/*.proto ../api/proto/services/*.proto

# Copy discord-client source code
COPY discord-client ./

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o gambler .

# Production stage
FROM alpine:latest AS prod

RUN apk --no-cache add ca-certificates tzdata

WORKDIR /root/

# Copy the binary from builder to a location in PATH
COPY --from=builder /app/gambler /usr/local/bin/gambler

# Run the binary
CMD ["gambler"]