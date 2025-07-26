#!/bin/bash

# Build lightweight server-only version of abaper
# This creates a REST API server without CLI dependencies

set -e

echo "🔧 Building lightweight abaper server (AI-only, no CLI/ADT dependencies)..."

# Build variables
BUILD_MODE="${BUILD_MODE:-production}"
VERSION="${VERSION:-v0.1.0-server-lite}"
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")

# Set build flags for server-only mode
LDFLAGS="-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME} -X main.GitCommit=${GIT_COMMIT} -X main.BuildMode=${BUILD_MODE}"

# Server-only build (minimal dependencies)
echo "📦 Building server-only binary..."
go build -ldflags "${LDFLAGS}" -o abaper-server .

# Create systemd service file for server deployment
cat > abaper-server.service << 'EOF'
[Unit]
Description=Abaper AI Server
After=network.target

[Service]
Type=simple
User=abaper
WorkingDirectory=/opt/abaper
ExecStart=/opt/abaper/abaper-server --server-only --log-file=/var/log/abaper/server.log
Restart=always
RestartSec=5

# Environment
Environment=PORT=8013

# Security
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
EOF

# Create Docker setup for lightweight deployment
cat > Dockerfile.server << 'EOF'
# Lightweight Docker image for server-only mode
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags "-X main.Version=v0.1.0-docker -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S_UTC') -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.BuildMode=production" -o abaper-server .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/abaper-server .

# Create logs directory
RUN mkdir -p /var/log/abaper

# Expose port
EXPOSE 8013

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8013/health || exit 1

CMD ["./abaper-server", "--server-only", "--log-file=/var/log/abaper/server.log"]
EOF

# Create docker-compose for easy deployment
cat > docker-compose.server.yml << 'EOF'
version: '3.8'

services:
  abaper-server:
    build:
      context: .
      dockerfile: Dockerfile.server
    ports:
      - "8013:8013"
    environment:
      - PORT=8013
    volumes:
      - ./logs:/var/log/abaper
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8013/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s

  # Optional: Add nginx reverse proxy
  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx.conf:/etc/nginx/nginx.conf:ro
      - ./ssl:/etc/nginx/ssl:ro
    depends_on:
      - abaper-server
    restart: unless-stopped
EOF

# Create nginx config for reverse proxy
cat > nginx.conf << 'EOF'
events {
    worker_connections 1024;
}

http {
    upstream abaper {
        server abaper-server:8013;
    }

    server {
        listen 80;
        server_name _;

        # Security headers
        add_header X-Frame-Options DENY;
        add_header X-Content-Type-Options nosniff;
        add_header X-XSS-Protection "1; mode=block";

        # Streaming support
        proxy_buffering off;
        proxy_cache off;

        location / {
            proxy_pass http://abaper;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
            
            # Streaming headers
            proxy_set_header Connection '';
            proxy_http_version 1.1;
            chunked_transfer_encoding off;
        }

        # Health check endpoint
        location /health {
            proxy_pass http://abaper;
            access_log off;
        }
    }
}
EOF

# Create startup script
cat > start-server.sh << 'EOF'
#!/bin/bash

# Start abaper in server-only mode
# Usage: ./start-server.sh [port]

PORT="${1:-8013}"

echo "🚀 Starting abaper server-only mode on port $PORT..."

# Create logs directory
mkdir -p ./logs

# Start server
./abaper-server --server-only --port "$PORT" --log-file ./logs/server.log
EOF

chmod +x start-server.sh

echo ""
echo "✅ Lightweight server build complete!"
echo ""
echo "📦 Files created:"
echo "  abaper-server              - Server-only binary"
echo "  abaper-server.service      - Systemd service file"
echo "  Dockerfile.server          - Lightweight Docker image"
echo "  docker-compose.server.yml  - Docker Compose setup"
echo "  nginx.conf                 - Nginx reverse proxy config"
echo "  start-server.sh            - Simple startup script"
echo ""
echo "🚀 Quick start options:"
echo ""
echo "1. 📱 Simple local server:"
echo "   ./start-server.sh"
echo ""
echo "2. 🐳 Docker deployment:"
echo "   docker-compose -f docker-compose.server.yml up -d"
echo ""
echo "3. 🖥️  System service:"
echo "   sudo cp abaper-server.service /etc/systemd/system/"
echo "   sudo systemctl enable abaper-server"
echo "   sudo systemctl start abaper-server"
echo ""
echo "🌐 Server-only endpoints:"
echo "  POST /api/v1/ai/generate    - AI text generation"
echo "  POST /api/v1/ai/chat        - Conversational AI"
echo "  GET  /health                - Health check"
echo "  GET  /version               - Version info"
echo ""
echo "🧪 Test streaming:"
echo "  curl \"http://localhost:8013/api/v1/system/stream-test?stream=true\""
echo ""
echo "✨ This server-only build has no CLI or SAP dependencies!"
