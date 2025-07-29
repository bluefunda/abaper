# ABAPER - POSIX Compliant ABAP Development Tool

A powerful, POSIX-compliant CLI and REST API tool that combines SAP ABAP Development Tools (ADT) REST services.

## ✨ Features

### 🚀 **Dual Mode Operation**
- **CLI Mode**: POSIX-compliant command-line interface for developers
- **Server Mode**: REST API for web applications and integrations

### ⚡ **High Performance**
- **Connection Caching**: 5-10x faster subsequent operations (30min cache)
- **Smart Session Management**: Automatic connection reuse and cleanup
- **Lightweight Ping Tests**: Validates cached connections efficiently

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

## 📦 **Installation**

### **Homebrew (macOS)**
```bash
brew install bluefunda/tap/abaper
```

### **Docker**
```bash
# Pull latest image
docker pull bluefunda/abaper:latest

# Run CLI
docker run --rm bluefunda/abaper:latest abaper --version

# Run server
docker run -p 8013:8013 bluefunda/abaper:latest abaper --server
```

### **Manual Installation**

Download pre-built binaries from the [releases page](https://github.com/bluefunda/abaper/releases):

#### **Linux**
```bash
# AMD64
curl -L https://github.com/bluefunda/abaper/releases/latest/download/abaper_Linux_x86_64.tar.gz | tar xz

# ARM64 (e.g., Raspberry Pi 4, AWS Graviton)
curl -L https://github.com/bluefunda/abaper/releases/latest/download/abaper_Linux_arm64.tar.gz | tar xz

# ARMv7 (e.g., Raspberry Pi 3)
curl -L https://github.com/bluefunda/abaper/releases/latest/download/abaper_Linux_armv7.tar.gz | tar xz
```

#### **macOS**
```bash
# Intel Macs
curl -L https://github.com/bluefunda/abaper/releases/latest/download/abaper_Darwin_x86_64.tar.gz | tar xz

# Apple Silicon Macs
curl -L https://github.com/bluefunda/abaper/releases/latest/download/abaper_Darwin_arm64.tar.gz | tar xz
```

#### **Windows**
Download `abaper_Windows_x86_64.zip` from the releases page and extract.

#### **Linux Packages**
```bash
# Debian/Ubuntu
wget https://github.com/bluefunda/abaper/releases/latest/download/abaper_amd64.deb
sudo dpkg -i abaper_amd64.deb

# Red Hat/CentOS/Fedora
wget https://github.com/bluefunda/abaper/releases/latest/download/abaper_amd64.rpm
sudo rpm -i abaper_amd64.rpm

# Alpine
wget https://github.com/bluefunda/abaper/releases/latest/download/abaper_amd64.apk
sudo apk add --allow-untrusted abaper_amd64.apk
```

## 🚀 **Quick Start**

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
abaper connect

# Get objects (uses cached connection - fast!)
abaper get program ZTEST
abaper get class ZCL_UTILITY
abaper get function RFC_READ_TABLE SRFC

# AI analysis
abaper analyze program ZTEST
abaper review class ZCL_UTILITY
abaper optimize function MY_FUNC MY_GROUP

# Search and discovery
abaper search objects "Z*"
abaper list packages

# Get help
abaper help
abaper help get
```

### **Logging Examples**

```bash
# Default quiet mode (minimal output)
abaper get program ZTEST

# Quiet mode with comprehensive file logging
abaper --log-file=./logs/abaper.log get program ZTEST

# Normal mode with standard CLI output
abaper --normal analyze class ZCL_TEST

# Verbose debugging with file logging
abaper --verbose --log-file=./debug.log connect

# Use environment variable for consistent logging
export ABAPER_LOG_FILE="./logs/abaper-$(date +%Y%m%d).log"
abaper get program ZTEST
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
time abaper connect                    # ~4 seconds

# Subsequent commands are lightning fast
time abaper get program ZTEST          # ~0.6 seconds
time abaper get class ZCL_TEST         # ~0.5 seconds
time abaper list packages              # ~0.5 seconds
time abaper analyze program ZTEST      # ~0.5s + AI time
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
abaper get program ZTEST
abaper get program Z_REPORT_SALES

# Classes
abaper get class ZCL_UTILITY_HELPER
abaper get class CL_STANDARD_CLASS

# Functions (requires function group)
abaper get function Z_CALCULATE_TAX Z_TAX_GROUP
abaper get function RFC_READ_TABLE SRFC

# Other objects
abaper get include ZINCLUDE_TOP
abaper get interface ZIF_BUSINESS_LOGIC
abaper get structure ZSTR_CUSTOMER_DATA
abaper get table ZTABLE_PRODUCTS
abaper get package $TMP
```

### **AI-Powered Analysis**
```bash
# Code analysis
abaper analyze program ZTEST
abaper analyze class ZCL_SALES_PROCESSOR

# Code review
abaper review program Z_FINANCIAL_REPORT
abaper review class ZCL_INTEGRATION_HANDLER

# Performance optimization
abaper optimize program Z_BATCH_PROCESSOR
abaper optimize function Z_HEAVY_CALCULATION Z_MATH_GROUP
```

### **Search and Discovery**
```bash
# Search objects by pattern
abaper search objects "Z*"
abaper search objects "CL_SALES*" class
abaper search objects "*INTEGRATION*" program class

# List packages
abaper list packages
abaper list packages "Z*"
abaper list packages "*DEV*"
```

### **Object Creation**
```bash
# Create with AI assistance
abaper create program Z_NEW_REPORT "Sales report with customer analysis"
abaper create class ZCL_NEW_SERVICE "RESTful service for customer data"
```

### **System Operations**
```bash
# Test connection
abaper connect

# Get help
abaper help                    # General help
abaper help get               # Help for 'get' command
abaper help analyze           # Help for 'analyze' command
```

## 🖥️ **Server Mode**

### **Start Server**
```bash
# Default port 8013
abaper --server

# Custom port
abaper --server -p 9000

# Quiet mode with file logging
abaper --server --log-file=./logs/server.log
```

### **REST API Endpoints**
- `POST /generate-code` - Code generation
- `POST /generate-code-stream` - Streaming code generation
- `GET /health` - Health check
- `GET /version` - Version information

### **Docker Support**

For Docker deployment examples, see [`examples/docker/`](examples/docker/).

```bash
# Quick start with published image
docker run -p 8013:8013 \
  -e SAP_HOST="your-host:8000" \
  -e SAP_USERNAME="your-user" \
  -e SAP_PASSWORD="your-password" \
  bluefunda/abaper:latest \
  abaper --server
```

## 🛠️ **Development**

### **Building from Source**

```bash
# Clone repository
git clone https://github.com/bluefunda/abaper
cd abaper

# Build for development
go build -o abaper .

# Run tests
go test ./...

# Format code
go fmt ./...
```

### **Testing Release Build**

```bash
# Test full release build locally (without publishing)
goreleaser build --snapshot --clean

# Test release process (dry run)
goreleaser release --snapshot --clean
```

### **Development Workflow**
```bash
# Quick development builds
go run . --version
go run . connect

# Format and validate
go fmt ./...
go vet ./...

# Run tests
go test ./...

# Build optimized binary
go build -ldflags="-s -w" -o abaper .
```

## 🔍 **Troubleshooting**

### **Connection Issues**
```bash
# Test basic connectivity
abaper connect

# Verbose logging to see cache behavior
abaper --verbose --log-file=./debug.log get program ZTEST

# Check credentials
echo $SAP_HOST $SAP_USERNAME
```

### **Common Problems**
1. **"ADT host not configured"**: Set `SAP_HOST` environment variable
2. **"Authentication failed"**: Verify `SAP_USERNAME` and `SAP_PASSWORD`
3. **"SICF services not found"**: Activate ADT services in transaction SICF
4. **Cache issues**: Cache automatically expires after 30 minutes

### **Debug Logging**
```bash
# See detailed logs including cache hits/misses
abaper --verbose --log-file=./debug.log connect
abaper --verbose --log-file=./debug.log get program ZTEST

# Cache status information (in log file)
{"level":"debug","msg":"Using cached ADT client","cache_age":"5m30s"}
{"level":"info","msg":"ADT cache miss","cache_expired":true}
```

## 🚀 **Release Process**

Releases are automated via GitHub Actions and GoReleaser when you push a tag:

```bash
# Create and push a new release
git tag v1.0.0
git push origin v1.0.0

# GitHub Actions automatically:
# - Builds for all supported platforms
# - Creates GitHub release with binaries
# - Updates Homebrew formula
# - Publishes Docker images
# - Generates checksums and changelogs
```

### **Supported Platforms**
- **Linux**: x86_64, ARM64, ARMv6, ARMv7
- **macOS**: x86_64 (Intel), ARM64 (Apple Silicon)
- **Windows**: x86_64
- **FreeBSD**: x86_64

### **Package Formats**
- **Archives**: tar.gz (Unix), zip (Windows)
- **Linux Packages**: DEB, RPM, APK
- **Package Managers**: Homebrew (macOS)
- **Container Images**: Multi-architecture Docker images

## 📁 **Project Structure**

```
abaper/
├── main.go              # Main application entry point
├── cli.go               # Command-line interface handlers
├── adt_client.go        # SAP ADT client implementation
├── rest/                # REST API server components
├── .goreleaser.yml      # GoReleaser configuration
├── .github/workflows/   # GitHub Actions CI/CD
│   └── release.yml      # Automated release workflow
├── examples/            # Usage examples
│   └── docker/          # Docker deployment examples
├── Dockerfile           # Container image definition
├── go.mod               # Go module definition
├── go.sum               # Go dependencies
├── LICENSE              # Apache 2.0 license
└── README.md            # This documentation
```

## 🤝 **Contributing**

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/new-feature`
3. Make your changes following POSIX standards
4. Test thoroughly: `go test ./...`
5. Test builds: `go build -o abaper .`
6. Commit: `git commit -am 'Add new feature'`
7. Push: `git push origin feature/new-feature`
8. Create a Pull Request

### **Development Guidelines**
- Use standard Go tooling (`go build`, `go test`, `go fmt`)
- Ensure POSIX compliance for all new features
- Add appropriate tests and documentation
- Test file logging functionality with `--log-file` option

## 📄 **License**

Copyright (c) 2025 BlueFunda, Inc. All rights reserved.

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) for details.

## 🔗 **Links**

- **Repository**: https://github.com/bluefunda/abaper
- **Releases**: https://github.com/bluefunda/abaper/releases
- **Docker Images**: https://hub.docker.com/r/bluefunda/abaper
- **Issues**: https://github.com/bluefunda/abaper/issues
- **Organization**: BlueFunda, Inc.

---

**Built with ❤️ by BlueFunda, Inc. - Making ABAP development faster, smarter, and more secure.**
