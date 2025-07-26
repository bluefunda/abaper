#!/bin/bash

# Clean build script for AI-free abaper
set -e

echo "🧹 Building clean AI-free abaper..."

# Clean previous builds
echo "1. Cleaning previous builds..."
rm -f abaper abaper-server abaper-test

# Verify Go files
echo "2. Verifying Go files..."
go fmt ./...
go vet ./...

# Download and tidy dependencies  
echo "3. Managing dependencies..."
go mod download
go mod tidy

# Test compilation
echo "4. Testing compilation..."
go build -o abaper-test .
if [ $? -eq 0 ]; then
    echo "✅ Test compilation successful"
    rm abaper-test
else
    echo "❌ Test compilation failed"
    exit 1
fi

# Build final binary
echo "5. Building final binary..."
go build -ldflags="-s -w -X main.Version=v0.0.1-no-ai -X main.BuildTime=$(date -u +%Y%m%d.%H%M%S) -X main.GitCommit=$(git rev-parse --short HEAD 2>/dev/null || echo 'unknown') -X main.BuildMode=release" -o abaper .

# Verify binary
if [ -f "abaper" ]; then
    echo "✅ Build successful!"
    echo "📦 Binary: ./abaper"
    echo "🔧 Version: $(./abaper --version)"
    echo "📏 Size: $(ls -lh abaper | awk '{print $5}')"
else
    echo "❌ Build failed - binary not created"
    exit 1
fi

echo ""
echo "🎉 Clean build complete!"
echo "Run './abaper --help' to get started"
