#!/bin/bash

# Enhanced build script for POSIX-compliant ABAPER with garble obfuscation
# This builds the new POSIX-compliant version with code obfuscation

set -e

VERSION="v0.0.1"
BUILD_TIME=$(date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_MODE="${BUILD_MODE:-dev}"

# Check if garble is installed, install if missing
check_and_install_garble() {
    if ! command -v garble &> /dev/null; then
        echo "⚠️  garble not found. Installing..."
        go install mvdan.cc/garble@latest

        if ! command -v garble &> /dev/null; then
            echo "❌ Failed to install garble. Please install manually:"
            echo "   go install mvdan.cc/garble@latest"
            echo "   Make sure \$GOPATH/bin is in your \$PATH"
            exit 1
        fi
        echo "✅ garble installed successfully"
    fi
    echo "🔧 Using garble version: $(garble version 2>/dev/null || echo 'unknown')"
}

echo "🚀 Building ABAPER ${VERSION}"
echo "Build time: ${BUILD_TIME}"
echo "Git commit: ${GIT_COMMIT}"
echo "Build mode: ${BUILD_MODE}"

# Set build flags
LDFLAGS="-X 'main.Version=${VERSION}' -X 'main.BuildTime=${BUILD_TIME}' -X 'main.GitCommit=${GIT_COMMIT}' -X 'main.BuildMode=${BUILD_MODE}'"

# Determine build approach based on mode
case "${BUILD_MODE}" in
    "release")
        echo ""
        echo "🔐 Building with FULL garble obfuscation (release mode)..."
        check_and_install_garble

        # Aggressive obfuscation + optimization
        LDFLAGS="${LDFLAGS} -s -w"
        export CGO_ENABLED=0

        echo "🔨 Running garble with full obfuscation..."
        garble -literals -tiny -seed=random build \
            -ldflags "${LDFLAGS}" \
            -trimpath \
            -o abaper .
        ;;

    "dev")
        echo ""
        echo "🔧 Building with MINIMAL garble obfuscation (dev mode)..."
        check_and_install_garble

        # Create debug directory for garble
        mkdir -p ./build/debug

        echo "🔨 Running garble with minimal obfuscation..."
        garble -debugdir=./build/debug build \
            -ldflags "${LDFLAGS}" \
            -o abaper .

        echo "📋 Debug info saved to: ./build/debug"
        ;;

    "debug")
        echo ""
        echo "🐛 Building WITHOUT obfuscation (debug mode)..."
        echo "⚠️  Skipping garble for maximum debugging capability"

        echo "🔨 Running standard go build..."
        go build -ldflags "${LDFLAGS}" -o abaper .
        ;;

    *)
        echo "❌ Invalid build mode: ${BUILD_MODE}"
        echo "   Valid modes: release, dev, debug"
        echo "   Example: BUILD_MODE=release ./build.sh"
        exit 1
        ;;
esac

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ Build successful!"
    echo ""
    echo "📋 Binary information:"
    ls -la abaper

    # Show file type and size
    if command -v file &> /dev/null; then
        echo "File type: $(file abaper)"
    fi

    # Check if symbols are stripped (for release builds)
    if [ "${BUILD_MODE}" = "release" ] && command -v nm &> /dev/null; then
        if nm abaper &>/dev/null; then
            echo "⚠️  Binary contains symbols"
        else
            echo "✅ Binary is stripped (symbols removed)"
        fi
    fi

    echo ""
    echo "🧪 Testing binary..."
    ./abaper --version
    echo ""
    echo "📖 POSIX Command Examples:"
    echo "  ./abaper get program ZTEST"
    echo "  ./abaper analyze class ZCL_TEST"
    echo "  ./abaper search objects \"Z*\""
    echo "  ./abaper list packages"
    echo "  ./abaper connect"
    echo ""
    echo "⚡ Performance Features:"
    echo "  ✅ ADT Connection Caching (30min timeout)"
    echo "  ✅ Session Reuse - Subsequent commands are 5-10x faster"
    echo "  ✅ Automatic cleanup on exit"

    case "${BUILD_MODE}" in
        "release")
            echo "  ✅ Full code obfuscation with garble"
            echo "  ✅ Binary optimization (-s -w flags)"
            ;;
        "dev")
            echo "  ✅ Minimal code obfuscation with garble"
            echo "  ✅ Debug info preserved"
            ;;
        "debug")
            echo "  ✅ No obfuscation for maximum debugging"
            echo "  ✅ Full symbol information available"
            ;;
    esac

    echo ""
    echo "🎉 POSIX-compliant ABAPER ready!"

    # Show build mode specific tips
    case "${BUILD_MODE}" in
        "release")
            echo ""
            echo "💡 Release Build Tips:"
            echo "  • Binary is fully obfuscated and optimized"
            echo "  • Use for production deployments"
            echo "  • Debugging will be limited due to obfuscation"
            ;;
        "dev")
            echo ""
            echo "💡 Development Build Tips:"
            echo "  • Minimal obfuscation preserves debugging"
            echo "  • Debug symbols saved in ./build/debug/"
            echo "  • Good balance of security and debuggability"
            ;;
        "debug")
            echo ""
            echo "💡 Debug Build Tips:"
            echo "  • No obfuscation - maximum debugging capability"
            echo "  • Use for development and troubleshooting"
            echo "  • Not recommended for production"
            ;;
    esac

else
    echo "❌ Build failed!"
    exit 1
fi

# Optional: Show quick build mode switching examples
echo ""
echo "🔄 Quick Build Mode Switching:"
echo "  BUILD_MODE=release ./build.sh  # Full obfuscation"
echo "  BUILD_MODE=dev ./build.sh      # Minimal obfuscation (default)"
echo "  BUILD_MODE=debug ./build.sh    # No obfuscation"
