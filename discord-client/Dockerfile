# Development stage
FROM golang:1.24-alpine AS dev

# Install development dependencies
RUN apk add --no-cache git

# Install air for hot reload
RUN go install github.com/air-verse/air@latest

# Ensure Go bin directory is in PATH
ENV PATH="/go/bin:${PATH}"

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

# Expose port if needed for debugging
EXPOSE 2345

CMD ["air"]

# Production builder stage
FROM golang:1.24-alpine AS builder

# Install dependencies
RUN apk add --no-cache git

WORKDIR /app

# Copy go mod files
COPY go.mod go.sum* ./
RUN go mod download

# Copy source code
COPY . .

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