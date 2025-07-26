# Multi-stage Dockerfile for CLI for ABAP Development Tool
# Copyright (c) 2025 BlueFunda, Inc. All rights reserved.

# Build stage
FROM golang:1.24-alpine AS builder

# Set working directory
WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Set build variables
ARG VERSION=v2.0.0-docker
ARG BUILD_MODE=prod
ARG BUILD_TIME
ARG GIT_COMMIT

# Build the application
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -X main.BuildMode=${BUILD_MODE}" \
    -o abaper .

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN adduser -D -s /bin/sh abaper

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/abaper .

# Copy configuration template (if exists)
COPY --from=builder /app/config.json.template ./config.json.template 2>/dev/null || true

# Set ownership
RUN chown -R abaper:abaper /app

# Switch to non-root user
USER abaper

# Expose port
EXPOSE 8013

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8013/health || exit 1

# Default command (server mode)
CMD ["./abaper", "--server", "-p", "8013"]

# Labels
LABEL maintainer="BlueFunda, Inc. <info@bluefunda.com>" \
    version="${VERSION}" \
    description="CLI for ABAP Development Tool by BlueFunda, Inc." \
    org.opencontainers.image.title="CLI for ABAP ADT" \
    org.opencontainers.image.description="CLI for ABAP development tool by BlueFunda, Inc." \
    org.opencontainers.image.version="${VERSION}" \
    org.opencontainers.image.created="${BUILD_TIME}" \
    org.opencontainers.image.revision="${GIT_COMMIT}" \
    org.opencontainers.image.vendor="BlueFunda, Inc." \
    org.opencontainers.image.source="https://github.com/bluefunda/abaper" \
    org.opencontainers.image.url="https://github.com/bluefunda/abaper" \
    org.opencontainers.image.documentation="https://github.com/bluefunda/abaper/blob/main/README.md" \
    org.opencontainers.image.licenses="Apache 2.0"
