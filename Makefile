# Makefile for CLI for ABAP Development Tool
# Copyright (c) 2025 BlueFunda, Inc. All rights reserved.
# Makefile for ABAPER with garble obfuscation support

# Configuration
PROJECT_NAME := abaper
VERSION := v0.0.1
BUILD_TIME := $(shell date -u '+%Y-%m-%d %H:%M:%S UTC')
GIT_COMMIT := $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_MODE ?= dev

# Directories
BUILD_DIR := ./build
DIST_DIR := ./dist
DEBUG_DIR := $(BUILD_DIR)/debug

# Go settings
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)

# Build flags
LDFLAGS := -X 'main.Version=$(VERSION)' \
           -X 'main.BuildTime=$(BUILD_TIME)' \
           -X 'main.GitCommit=$(GIT_COMMIT)' \
           -X 'main.BuildMode=$(BUILD_MODE)'

# Output binary
BINARY := $(PROJECT_NAME)
ifeq ($(GOOS),windows)
	BINARY := $(PROJECT_NAME).exe
endif

# Colors for output
GREEN := \033[32m
YELLOW := \033[33m
BLUE := \033[34m
CYAN := \033[36m
RED := \033[31m
NC := \033[0m

.PHONY: all build release dev debug clean install garble-install test fmt vet lint deps dist help

# Default target
all: dev

# Help target
help:
	@echo "$(CYAN)ABAPER Build System with Garble Obfuscation$(NC)"
	@echo ""
	@echo "$(YELLOW)Available targets:$(NC)"
	@echo "  $(GREEN)build$(NC)          Build binary (same as dev)"
	@echo "  $(GREEN)dev$(NC)            Build development binary with minimal obfuscation"
	@echo "  $(GREEN)release$(NC)        Build release binary with full obfuscation"
	@echo "  $(GREEN)debug$(NC)          Build debug binary without obfuscation"
	@echo "  $(GREEN)clean$(NC)          Clean build artifacts"
	@echo "  $(GREEN)install$(NC)        Install binary to GOPATH/bin"
	@echo "  $(GREEN)garble-install$(NC) Install garble obfuscation tool"
	@echo "  $(GREEN)test$(NC)           Run tests"
	@echo "  $(GREEN)fmt$(NC)            Format code"
	@echo "  $(GREEN)vet$(NC)            Run go vet"
	@echo "  $(GREEN)lint$(NC)           Run golangci-lint"
	@echo "  $(GREEN)deps$(NC)           Download dependencies"
	@echo "  $(GREEN)dist$(NC)           Create distribution packages"
	@echo "  $(GREEN)help$(NC)           Show this help message"
	@echo ""
	@echo "$(YELLOW)Examples:$(NC)"
	@echo "  make release        # Build optimized release with full obfuscation"
	@echo "  make dev            # Build development version"
	@echo "  make clean release  # Clean and build release"
	@echo "  make GOOS=linux GOARCH=amd64 release  # Cross-compile"

# Install garble if not present
garble-install:
	@if ! command -v garble >/dev/null 2>&1; then \
		echo "$(YELLOW)Installing garble...$(NC)"; \
		go install mvdan.cc/garble@latest; \
		echo "$(GREEN)Garble installed successfully$(NC)"; \
	else \
		echo "$(GREEN)Garble already installed: $$(garble version)$(NC)"; \
	fi

# Create build directories
$(BUILD_DIR) $(DIST_DIR) $(DEBUG_DIR):
	@mkdir -p $@

# Development build with minimal obfuscation
dev: garble-install | $(BUILD_DIR) $(DEBUG_DIR)
	@echo "$(BLUE)Building development binary with minimal garble obfuscation...$(NC)"
	@export CGO_ENABLED=0 && \
	 garble -debugdir=$(DEBUG_DIR) build -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo "$(GREEN)Development build completed: $(BINARY)$(NC)"
	@echo "$(YELLOW)Debug info saved to: $(DEBUG_DIR)$(NC)"

# Alias for dev
build: dev

# Release build with full obfuscation
release: garble-install | $(BUILD_DIR)
	@echo "$(BLUE)Building release binary with full garble obfuscation...$(NC)"
	@export CGO_ENABLED=0 && \
	 export GOOS=$(GOOS) && \
	 export GOARCH=$(GOARCH) && \
	 garble -literals -tiny -seed=random build \
	   -ldflags "$(LDFLAGS) -s -w" \
	   -trimpath \
	   -o $(BUILD_DIR)/$(BINARY) .
	@cp $(BUILD_DIR)/$(BINARY) $(BINARY)
	@echo "$(GREEN)Release build completed: $(BINARY)$(NC)"

# Debug build without obfuscation
debug: | $(BUILD_DIR)
	@echo "$(BLUE)Building debug binary without obfuscation...$(NC)"
	@go build -ldflags "$(LDFLAGS)" -o $(BINARY) .
	@echo "$(GREEN)Debug build completed: $(BINARY)$(NC)"
	@echo "$(YELLOW)No obfuscation applied for maximum debugging capability$(NC)"

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR) $(DIST_DIR)
	@rm -f $(BINARY)
	@echo "$(GREEN)Clean completed$(NC)"

# Install to GOPATH/bin
install: release
	@echo "$(BLUE)Installing $(PROJECT_NAME) to GOPATH/bin...$(NC)"
	@go install -ldflags "$(LDFLAGS) -s -w" .
	@echo "$(GREEN)Installation completed$(NC)"

# Run tests
test:
	@echo "$(BLUE)Running tests...$(NC)"
	@go test -v ./...

# Format code
fmt:
	@echo "$(BLUE)Formatting code...$(NC)"
	@go fmt ./...
	@echo "$(GREEN)Code formatted$(NC)"

# Run go vet
vet:
	@echo "$(BLUE)Running go vet...$(NC)"
	@go vet ./...
	@echo "$(GREEN)Vet completed$(NC)"

# Run linter (requires golangci-lint)
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "$(BLUE)Running golangci-lint...$(NC)"; \
		golangci-lint run; \
		echo "$(GREEN)Linting completed$(NC)"; \
	else \
		echo "$(YELLOW)golangci-lint not found, skipping...$(NC)"; \
		echo "Install with: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Download dependencies
deps:
	@echo "$(BLUE)Downloading dependencies...$(NC)"
	@go mod download
	@go mod tidy
	@echo "$(GREEN)Dependencies updated$(NC)"

# Create distribution packages
dist: release | $(DIST_DIR)
	@echo "$(BLUE)Creating distribution packages...$(NC)"
	@DIST_NAME="$(PROJECT_NAME)_$(VERSION)_$(GOOS)_$(GOARCH)" && \
	 DIST_PATH="$(DIST_DIR)/$$DIST_NAME" && \
	 mkdir -p "$$DIST_PATH" && \
	 cp $(BUILD_DIR)/$(BINARY) "$$DIST_PATH/" && \
	 echo "$(PROJECT_NAME) $(VERSION)" > "$$DIST_PATH/README.txt" && \
	 echo "Built: $(BUILD_TIME)" >> "$$DIST_PATH/README.txt" && \
	 echo "Commit: $(GIT_COMMIT)" >> "$$DIST_PATH/README.txt" && \
	 echo "Target: $(GOOS)/$(GOARCH)" >> "$$DIST_PATH/README.txt" && \
	 echo "Obfuscated with garble" >> "$$DIST_PATH/README.txt" && \
	 cd $(DIST_DIR) && \
	 tar -czf "$$DIST_NAME.tar.gz" "$$DIST_NAME" && \
	 cd .. && \
	 echo "$(GREEN)Distribution created: $(DIST_DIR)/$$DIST_NAME.tar.gz$(NC)"

# Cross-compile targets
linux:
	@$(MAKE) release GOOS=linux GOARCH=amd64

darwin:
	@$(MAKE) release GOOS=darwin GOARCH=amd64

windows:
	@$(MAKE) release GOOS=windows GOARCH=amd64

# Multi-platform distribution
dist-all: clean
	@echo "$(BLUE)Creating multi-platform distributions...$(NC)"
	@$(MAKE) dist GOOS=linux GOARCH=amd64
	@$(MAKE) dist GOOS=darwin GOARCH=amd64
	@$(MAKE) dist GOOS=windows GOARCH=amd64
	@echo "$(GREEN)All distributions created in $(DIST_DIR)/$(NC)"
	@ls -la $(DIST_DIR)/*.tar.gz

# Quick check target
check: fmt vet test
	@echo "$(GREEN)All checks passed$(NC)"
