# Copyright (c) 2025 BlueFunda, Inc. All rights reserved.
# Simplified Dockerfile for GoReleaser
# GoReleaser handles the building, this just creates the runtime container

FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata curl

# Create non-root user
RUN adduser -D -s /bin/sh abaper

# Set working directory
WORKDIR /app

# Copy binary (GoReleaser will place the built binary here)
COPY abaper .

# Set ownership
RUN chown -R abaper:abaper /app

# Switch to non-root user
USER abaper

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=30s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command (server mode)
CMD ["./abaper", "--server", "-p", "8080"]

# Labels will be added by GoReleaser build flags
