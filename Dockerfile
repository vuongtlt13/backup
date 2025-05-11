# Build stage
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application
RUN CGO_ENABLED=0 GOOS=linux go build -o backupdb

# Final stage
FROM alpine:3.19

# Install runtime dependencies
RUN apk update && apk add --no-cache \
    ca-certificates \
    tzdata \
    openssh \
    rsync \
    bash

# Create necessary directories
RUN mkdir -p /app/backups /app/config && \
    chown -R 999:0 /app && chmod -R 755 /app

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/backupdb /app/
COPY --from=builder /app/config.yaml.example /app/config/config.yaml

# Set environment variables
ENV TZ=UTC

# Run the application
ENTRYPOINT ["/app/backupdb"]
CMD ["--config", "/app/config/config.yaml"] 