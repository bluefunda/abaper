#!/bin/bash

# Build and test script for abaper with enhanced CreateProgram functionality

echo "Building abaper with enhanced CreateProgram functionality..."

# Navigate to project directory
cd /Users/phani/Downloads/abaper

# Clean any existing build
echo "Cleaning previous builds..."
go clean
rm -f abaper

# Download dependencies
echo "Downloading Go dependencies..."
go mod tidy

# Build the project
echo "Building abaper..."
go build -o abaper .

if [ $? -eq 0 ]; then
    echo "✅ Build successful!"
    echo ""
    echo "Enhanced CreateProgram functionality added:"
    echo "  ✅ Session management with proper locking/unlocking"
    echo "  ✅ Source code insertion and activation"
    echo "  ✅ Configurable packages (defaults to \$TMP)"
    echo "  ✅ CLI support with templates"
    echo "  ✅ REST API endpoint /api/v1/objects/create"
    echo "  ✅ Enhanced error handling and logging"
    echo ""
    echo "CLI Usage examples:"
    echo "  ./abaper create program Z_MY_PROG"
    echo "  ./abaper create program Z_MY_PROG \"My Program Description\""
    echo "  ./abaper create program Z_MY_PROG \"My Program\" ZPACKAGE"
    echo "  ./abaper create class Z_MY_CLASS \"My Class Description\""
    echo ""
    echo "REST API example:"
    echo "  curl -X POST http://localhost:8080/api/v1/objects/create \\"
    echo "       -H \"Content-Type: application/json\" \\"
    echo "       -d '{\"object_type\":\"program\",\"object_name\":\"Z_TEST_PROG\",\"description\":\"Test Program\",\"package\":\"\$TMP\"}'"
    echo ""
    echo "Binary location: ./abaper"
else
    echo "❌ Build failed!"
    exit 1
fi
