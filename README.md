# ABAPER v0.0.1 - POSIX Compliant ABAP Development Tool

A powerful, POSIX-compliant CLI and REST API tool that combines SAP ABAP Development Tools (ADT) REST services.

## ✨ Features

### 🚀 **Dual Mode Operation**
- **CLI Mode**: POSIX-compliant command-line interface for developers
- **Server Mode**: REST API for web applications and integrations

### ⚡ **High Performance**
- **Connection Caching**: 5-10x faster subsequent operations (30min cache)
- **Smart Session Management**: Automatic connection reuse and cleanup
- **Lightweight Ping Tests**: Validates cached connections efficiently

### 🔒 **Security Features**
- **Code Obfuscation**: Binaries built with [garble](https://github.com/burrowers/garble) for enhanced security
- **Symbol Stripping**: Release binaries have debugging symbols removed
- **Literal Obfuscation**: String constants are obfuscated to prevent reverse engineering
- **Control Flow Obfuscation**: Makes static analysis significantly more difficult

### 🔧 **SAP ADT Integration**
- Connect to SAP systems via ADT REST services
- Retrieve ABAP programs, classes, functions, structures, tables, interfaces
- Search and browse SAP repository
- List packages and objects
- Create and update ABAP objects

### ✅ **POSIX Compliance**
- Standard exit codes (0, 1, 2, 130, 143)
- Proper signal handling (SIGINT, SIGTERM)
- Clean argument parsing
- Consistent error reporting

### 🔕 **Enhanced User Experience**
- **Quiet Mode Default**: Minimal CLI output for clean automation
- **File Logging**: Comprehensive logging to files with `--log-file` option
- **Flexible Output Modes**: Choose between quiet, normal, and verbose modes
- **Environment Variable Support**: Configure via `ABAPER_LOG_FILE` and other env vars

## 📖 **Quick Start**

### **Installation**

```bash
# Clone repository
git clone https://github.com/bluefunda/abaper
cd abaper

# Build with default settings (development mode)
chmod +x build.sh
./build.sh

# Test
chmod +x test.sh
./test.sh
```

### **Environment Setup**

```bash

# Required for SAP operations
export SAP_HOST="your-sap-host:8000"
export SAP_CLIENT="100"
export SAP_USERNAME="your-username"
export SAP_PASSWORD="your-password"

# Optional: Default log file
export ABAPER_LOG_FILE="./logs/abaper.log"
```

### **Basic Usage**

```bash
# Test connection (establishes cache)
./abaper connect

# Get objects (uses cached connection - fast!)
./abaper get program ZTEST
./abaper get class ZCL_UTILITY
./abaper get function RFC_READ_TABLE SRFC

# AI analysis
./abaper analyze program ZTEST
./abaper review class ZCL_UTILITY
./abaper optimize function MY_FUNC MY_GROUP

# Search and discovery
./abaper search objects "Z*"
./abaper list packages

# Get help
./abaper help
./abaper help get
```

### **Logging Examples**

```bash
# Default quiet mode (minimal output)
./abaper get program ZTEST

# Quiet mode with comprehensive file logging
./abaper --log-file=./logs/abaper.log get program ZTEST

# Normal mode with standard CLI output
./abaper --normal analyze class ZCL_TEST

# Verbose debugging with file logging
./abaper --verbose --log-file=./debug.log connect

# Use environment variable for consistent logging
export ABAPER_LOG_FILE="./logs/abaper-$(date +%Y%m%d).log"
./abaper get program ZTEST
```

## 🔨 **Build System**

ABAPER uses an enhanced build system with [garble](https://github.com/burrowers/garble) for code obfuscation, providing three distinct build modes for different use cases.

### **Build Modes**

#### **🔧 Development Mode (default)**
```bash
./build.sh
# or
BUILD_MODE=dev ./build.sh
```
- **Minimal obfuscation** with debugging info preserved
- **Debug symbols** saved in `./build/debug/`
- **Fast compilation** for development workflow
- **Good balance** of security and debuggability

#### **🚀 Release Mode (production)**
```bash
BUILD_MODE=release ./build.sh
```
- **Full garble obfuscation** with `-literals -tiny -seed=random`
- **Symbol stripping** (`-s -w`) for smaller binaries
- **Maximum security** against reverse engineering
- **Optimized performance** for production use

#### **🐛 Debug Mode (troubleshooting)**
```bash
BUILD_MODE=debug ./build.sh
```
- **No obfuscation** for maximum debugging capability
- **Full symbol information** available for debuggers
- **Ideal for development** and troubleshooting
- **Not recommended** for production deployment

### **Advanced Build Options**

#### **Using Makefile**
```bash
# Install garble (if not present)
make garble-install

# Quick builds
make build          # Development build (default)
make dev           # Development build with minimal obfuscation
make release       # Release build with full obfuscation
make debug         # Debug build without obfuscation

# Cross-platform builds
make linux         # Build for Linux AMD64
make darwin        # Build for macOS AMD64
make windows       # Build for Windows AMD64

# Multi-platform distribution
make dist-all      # Create distributions for all platforms

# Maintenance
make clean         # Clean build artifacts
make test          # Run tests
make fmt           # Format code
make vet           # Run go vet
make lint          # Run linter (requires golangci-lint)
```

#### **Supported Platforms**
- **Linux**: amd64, arm64
- **macOS**: amd64, arm64 (Intel & Apple Silicon)
- **Windows**: amd64

### **Build Security Features**

#### **Garble Obfuscation**
- **Automatic Installation**: Garble is installed automatically if not present
- **Literal Obfuscation**: String constants are scrambled
- **Control Flow Obfuscation**: Code structure is modified
- **Symbol Obfuscation**: Function and variable names are randomized
- **Import Path Obfuscation**: Package imports are obfuscated

#### **Binary Analysis**
The build system automatically analyzes binaries:
```bash
# Example output
✅ Build successful!
📋 Binary information:
-rwxr-xr-x  1 user  staff   8.2M Jul 22 10:30 abaper
File type: abaper: Mach-O 64-bit executable x86_64
✅ Binary is stripped (symbols removed)
```

## 📚 **Command Reference**

### **Syntax**
```bash
abaper [OPTIONS] ACTION TYPE [NAME] [ARGS...]
```

### **Actions**
- `get` - Retrieve ABAP object source code
- `search` - Search for ABAP objects
- `list` - List objects (packages, etc.)
- `connect` - Test ADT connection
- `help` - Show help information

### **Object Types**
- `program` - ABAP program/report
- `class` - ABAP class
- `function` - ABAP function module (requires function group)
- `include` - ABAP include
- `interface` - ABAP interface
- `structure` - ABAP structure
- `table` - ABAP table
- `package` - ABAP package

### **Options**
- `-h, --help` - Show help message
- `-v, --version` - Show version information
- `-q, --quiet` - Quiet mode (DEFAULT - minimal CLI output)
- `--normal` - Normal mode (show standard output)
- `-V, --verbose` - Enable verbose output with debug info
- `--log-file=FILE` - Log to specified file (auto-creates directory)
- `--server` - Run as REST API server
- `-p, --port PORT` - Port for server mode (default: 8013)
- `--adt-host=HOST` - SAP system host
- `--adt-client=CLIENT` - SAP client
- `--adt-username=USER` - SAP username
- `--adt-password=PASS` - SAP password

### **Exit Status**
- `0` - Success
- `1` - General error
- `2` - Invalid usage
- `130` - Interrupted by user (Ctrl+C)
- `143` - Terminated by signal

## 🔥 **Performance Features**

### **Connection Caching**
- **First Command**: ~3-5 seconds (authentication + cache)
- **Subsequent Commands**: ~0.5-1 seconds (uses cache!)
- **Cache Duration**: 30 minutes
- **Smart Validation**: Automatic ping tests and cleanup

### **Typical Workflow**
```bash
# First command establishes cache
time ./abaper connect                    # ~4 seconds

# Subsequent commands are lightning fast
time ./abaper get program ZTEST          # ~0.6 seconds
time ./abaper get class ZCL_TEST         # ~0.5 seconds
time ./abaper list packages              # ~0.5 seconds
time ./abaper analyze program ZTEST      # ~0.5s + AI time
```

### **Cache Management**
- Automatic expiration after 30 minutes
- Config change detection (different SAP system/credentials)
- Connection health monitoring via ping tests
- Graceful cleanup on exit

## 📋 **Detailed Examples**

### **Object Retrieval**
```bash
# Programs
./abaper get program ZTEST
./abaper get program Z_REPORT_SALES

# Classes
./abaper get class ZCL_UTILITY_HELPER
./abaper get class CL_STANDARD_CLASS

# Functions (requires function group)
./abaper get function Z_CALCULATE_TAX Z_TAX_GROUP
./abaper get function RFC_READ_TABLE SRFC

# Other objects
./abaper get include ZINCLUDE_TOP
./abaper get interface ZIF_BUSINESS_LOGIC
./abaper get structure ZSTR_CUSTOMER_DATA
./abaper get table ZTABLE_PRODUCTS
./abaper get package $TMP
```

### **AI-Powered Analysis**
```bash
# Code analysis
./abaper analyze program ZTEST
./abaper analyze class ZCL_SALES_PROCESSOR

# Code review
./abaper review program Z_FINANCIAL_REPORT
./abaper review class ZCL_INTEGRATION_HANDLER

# Performance optimization
./abaper optimize program Z_BATCH_PROCESSOR
./abaper optimize function Z_HEAVY_CALCULATION Z_MATH_GROUP
```

### **Search and Discovery**
```bash
# Search objects by pattern
./abaper search objects "Z*"
./abaper search objects "CL_SALES*" class
./abaper search objects "*INTEGRATION*" program class

# List packages
./abaper list packages
./abaper list packages "Z*"
./abaper list packages "*DEV*"
```

### **Object Creation**
```bash
# Create with AI assistance
./abaper create program Z_NEW_REPORT "Sales report with customer analysis"
./abaper create class ZCL_NEW_SERVICE "RESTful service for customer data"
```

### **System Operations**
```bash
# Test connection
./abaper connect

# Get help
./abaper help                    # General help
./abaper help get               # Help for 'get' command
./abaper help analyze           # Help for 'analyze' command
```

## 🖥️ **Server Mode**

### **Start Server**
```bash
# Default port 8013
./abaper --server

# Custom port
./abaper --server -p 9000

# Quiet mode with file logging
./abaper --server --log-file=./logs/server.log
```

### **REST API Endpoints**
- `POST /generate-code` - Code generation
- `POST /generate-code-stream` - Streaming code generation
- `GET /health` - Health check
- `GET /version` - Version information

### **Docker Support**
```bash
# Build image (uses multi-stage build with garble)
docker build -t abaper .

# Run container
docker run -p 8013:8013 \
  -e SAP_HOST="your-host:8000" \
  -e SAP_USERNAME="your-user" \
  -e SAP_PASSWORD="your-password" \
  abaper

# Docker Compose
docker-compose up -d
```

## 🔧 **Development**

### **Development Workflow**
```bash
# Quick development builds (minimal obfuscation)
./build.sh                      # or BUILD_MODE=dev ./build.sh

# Test changes
./abaper --version
./abaper connect

# Format and validate
make fmt vet

# Run tests
make test

# Production build (full obfuscation)
BUILD_MODE=release ./build.sh
```

### **Release Process**
```bash
# 1. Clean workspace
make clean

# 2. Run tests
make test

# 3. Build for all platforms
make dist-all

# 4. Verify binaries
ls -la build/*/abaper*
ls -la dist/*.tar.gz

# 5. Test release binary
./abaper --version
```

### **Build System Architecture**

```
Build System
├── build.sh              # Main build script with garble
├── Makefile              # Build automation with cross-platform support
├── .github/workflows/    # CI/CD pipeline
│   └── build.yml         # Automated builds and releases
├── build/                # Build artifacts
│   ├── linux_amd64/     # Platform-specific builds
│   ├── darwin_arm64/
│   └── debug/           # Debug information (dev mode)
└── dist/                # Distribution packages
    ├── abaper_v0.0.1_linux_amd64.tar.gz
    └── *.sha256         # Checksums
```


## 🔍 **Troubleshooting**

### **Connection Issues**
```bash
# Test basic connectivity
./abaper connect

# Verbose logging to see cache behavior
./abaper --verbose --log-file=./debug.log get program ZTEST

# Check credentials
echo $SAP_HOST $SAP_USERNAME
```

### **Build Issues**
```bash
# Build without obfuscation for debugging
BUILD_MODE=debug ./build.sh

# Check garble installation
garble version

# Manual garble installation
go install mvdan.cc/garble@latest

# Clean build
make clean
./build.sh
```

### **Logging Issues**
```bash
# Test file logging
./abaper --log-file=./test.log --verbose get program ZTEST

# Check log file permissions
ls -la ./test.log

# Use environment variable
export ABAPER_LOG_FILE="./logs/abaper.log"
./abaper get program ZTEST
```

### **Common Problems**
1. **"garble: command not found"**: Install garble or use debug mode
2. **"ADT host not configured"**: Set `SAP_HOST` environment variable
3. **"Authentication failed"**: Verify `SAP_USERNAME` and `SAP_PASSWORD`
4. **"SICF services not found"**: Activate ADT services in transaction SICF
5. **Cache issues**: Cache automatically expires after 30 minutes
6. **Build failures**: Try debug mode first, then check Go version
7. **"Log file creation failed"**: Check directory permissions or create manually

### **Debug Logging**
```bash
# See detailed logs including cache hits/misses
./abaper --verbose --log-file=./debug.log connect
./abaper --verbose --log-file=./debug.log get program ZTEST

# Cache status information (in log file)
{"level":"debug","msg":"Using cached ADT client","cache_age":"5m30s"}
{"level":"info","msg":"ADT cache miss","cache_expired":true}
```

### **Binary Analysis**
```bash
# Verify garble worked
file abaper                    # Should show stripped binary
nm abaper 2>&1 | grep -q "no symbols" && echo "Obfuscated" || echo "Not obfuscated"

# Compare binary sizes (obfuscated should be larger)
BUILD_MODE=debug ./build.sh && mv abaper abaper-debug
BUILD_MODE=release ./build.sh
ls -lh abaper*
```

## 📁 **Project Structure**

```
abaper/
├── main.go              # Enhanced main application with quiet mode + file logging
├── cli.go               # Clean command handlers
├── adt_client.go        # SAP ADT client with all object types
├── go.mod               # Go module definition
├── go.sum               # Go dependencies
├── build.sh             # Enhanced build script with garble
├── Makefile             # Build automation with cross-platform support
├── test.sh              # Test script
├── Dockerfile           # Docker configuration with multi-stage builds
├── docker-compose.yml   # Docker Compose setup
├── .github/             # GitHub Actions CI/CD
│   └── workflows/
│       └── build.yml    # Automated builds and releases
├── .gitignore           # Git ignore rules
├── main.go.backup       # Original main.go (backup)
└── README.md            # This documentation
```

## 🚀 **CI/CD Pipeline**

ABAPER includes a complete GitHub Actions workflow for automated building and releasing:

### **Automated Features**
- **Pull Request Builds**: Development builds with minimal obfuscation
- **Release Builds**: Full obfuscation for tagged releases
- **Multi-Platform**: Automatic builds for Linux, macOS, Windows (amd64, arm64)
- **Security Scanning**: Automated vulnerability and security analysis
- **Distribution**: Automatic release creation with binaries and checksums

### **Release Process**
```bash
# 1. Create and push tag
git tag v0.1.0
git push origin v0.1.0

# 2. GitHub Actions automatically:
#    - Runs tests and security scans
#    - Builds for all platforms with full obfuscation
#    - Creates release with distribution packages
#    - Generates SHA256 checksums
#    - Publishes to GitHub Releases
```

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes following POSIX standards
4. Test thoroughly: `./test.sh`
5. Test builds: `BUILD_MODE=debug ./build.sh && BUILD_MODE=release ./build.sh`
6. Commit: `git commit -am 'Add new feature'`
7. Push: `git push origin feature/new-feature`
8. Create a Pull Request

### **Development Guidelines**
- Use `BUILD_MODE=dev` or `BUILD_MODE=debug` for development
- Test with `BUILD_MODE=release` before submitting PRs
- Ensure POSIX compliance for all new features
- Add appropriate tests and documentation
- Test file logging functionality with `--log-file` option

## 📄 **License**

Copyright (c) 2025 BlueFunda, Inc. All rights reserved.

## 🔗 **Links**

- **Repository**: https://github.com/bluefunda/abaper
- **Organization**: BlueFunda, Inc.
- **Issues**: https://github.com/bluefunda/abaper/issues
- **Garble Project**: https://github.com/burrowers/garble

---

**Built with ❤️ by BlueFunda, Inc. - Making ABAP development faster, smarter, and more secure.**
