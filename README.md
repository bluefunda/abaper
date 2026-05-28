# ABAPer CLI

[![Apache 2.0](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/bluefunda/abaper)](https://goreportcard.com/report/github.com/bluefunda/abaper)
[![pkg.go.dev](https://pkg.go.dev/badge/github.com/bluefunda/abaper.svg)](https://pkg.go.dev/github.com/bluefunda/abaper)

Command line interface and Go SDK for the [ABAPer](https://abaper.bluefunda.com) platform — generate, deploy, and test ABAP objects directly from your terminal, and a reusable Go library for SAP ABAP development tooling.

## Installation

### One-line installer (macOS and Linux)

```bash
sh -c "$(curl -fsSL https://raw.githubusercontent.com/bluefunda/abaper/main/install.sh)"
```

Installs to `/usr/local/bin` if writable, otherwise `~/.local/bin`. Override with `ABAPER_INSTALL_DIR`.

### Homebrew (macOS)

```bash
brew tap bluefunda/tap
brew install --cask abaper
```

### Debian / Ubuntu

```bash
VER=$(curl -fsSL https://api.github.com/repos/bluefunda/abaper/releases/latest | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -sL "https://github.com/bluefunda/abaper/releases/download/v${VER}/abaper_${VER}_linux_${ARCH}.deb" -o abaper.deb
sudo dpkg -i abaper.deb
```

### RHEL / Fedora / Rocky

```bash
VER=$(curl -fsSL https://api.github.com/repos/bluefunda/abaper/releases/latest | grep '"tag_name"' | sed 's/.*"v\([^"]*\)".*/\1/')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
sudo dnf install "https://github.com/bluefunda/abaper/releases/download/v${VER}/abaper_${VER}_linux_${ARCH}.rpm"
```

### From Source

```bash
go install github.com/bluefunda/abaper/cmd/abaper@latest
```

### Docker

```bash
docker pull bluefunda/abaper
docker run --rm bluefunda/abaper version
```

### From GitHub Releases

Download the binary for your platform from the [releases page](https://github.com/bluefunda/abaper/releases).

| Platform       | Archive                              |
|----------------|--------------------------------------|
| macOS (ARM64)  | `abaper_<version>_darwin_arm64.zip`  |
| macOS (AMD64)  | `abaper_<version>_darwin_amd64.zip`  |
| Linux (AMD64)  | `abaper_<version>_linux_amd64.tar.gz` |
| Linux (ARM64)  | `abaper_<version>_linux_arm64.tar.gz` |
| Windows (AMD64)| `abaper_<version>_windows_amd64.zip`  |
| Windows (ARM64)| `abaper_<version>_windows_arm64.zip`  |

## Quick Start

```bash
# Authenticate
abaper login

# Verify connection
abaper status

# Create and deploy an ABAP program
abaper generate --type program --name ZMY_REPORT
abaper deploy --type program --name ZMY_REPORT --source-file report.abap
```

## CLI Commands

### `abaper login`

Authenticate with the ABAPer platform using the OAuth2 device authorization flow. Opens a browser window for interactive login. Credentials are stored locally in `~/.abaper/tokens.yaml` with restricted permissions (0600).

```bash
abaper login
```

### `abaper logout`

Clear stored credentials by removing the local token file.

```bash
abaper logout
```

### `abaper status`

Show connection and authentication status. Reports the configured base URL, realm, organization, authentication state, and API health.

```bash
abaper status
abaper status -o json
```

### `abaper generate`

Create ABAP objects on the target SAP system. Supports programs, classes, and interfaces. Accepts source from a file or generates default templates.

```bash
# Generate with default template
abaper generate --type program --name ZMY_PROGRAM

# Generate from source file
abaper generate --type class --name ZCL_MY_CLASS --source-file my_class.abap

# Generate an interface
abaper generate --type interface --name ZIF_MY_INTERFACE
```

**Flags:**

| Flag            | Required | Default   | Description                        |
|-----------------|----------|-----------|------------------------------------|
| `--name`        | Yes      | —         | Object name                        |
| `--type`        | No       | `program` | Object type: program, class, interface |
| `--source-file` | No       | —         | Path to ABAP source file           |

### `abaper deploy`

Upload source code and activate an ABAP object in a single step.

```bash
abaper deploy --type program --name ZMY_PROGRAM --source-file program.abap
```

**Flags:**

| Flag            | Required | Default   | Description                        |
|-----------------|----------|-----------|------------------------------------|
| `--name`        | Yes      | —         | Object name                        |
| `--type`        | No       | `program` | Object type: program, class, interface |
| `--source-file` | Yes      | —         | Path to ABAP source file           |

### `abaper test`

Run ABAP unit tests for an object on the target SAP system.

```bash
abaper test --type class --name ZCL_MY_CLASS
abaper test --type class --name ZCL_MY_CLASS -o json
```

### `abaper list objects`

List ABAP objects, optionally filtered by package or type.

```bash
abaper list objects --package ZDEV
abaper list objects --type program
```

### `abaper list packages`

List contents of an ABAP package.

```bash
abaper list packages --name ZDEV
```

### `abaper ai chat`

Send a prompt to the ABAPer AI assistant and stream the response.

```bash
abaper ai chat "Explain SELECT FOR ALL ENTRIES in ABAP"
abaper ai chat "Optimize this code" --context-file program.abap
abaper ai chat "What about performance?" --chat-id <previous-chat-id>
```

### `abaper version`

Print the CLI version.

```bash
abaper version
```

## Global Flags

| Flag              | Description                                        |
|-------------------|----------------------------------------------------|
| `--base-url`      | ABAPer API base URL (default: `https://api.bluefunda.com`) |
| `--realm`         | Keycloak realm (default: `trm`)                    |
| `-o`, `--output`  | Output format: `text`, `json` (default: `text`)    |

## Configuration

Configuration is loaded in the following order of precedence:

1. **CLI flags** — `--base-url`, `--realm`
2. **Environment variables** — `ABAPER_BASE_URL`, `ABAPER_REALM`, `ABAPER_ORG`
3. **Config file** — `~/.abaper/config.yaml`

### Config File

```yaml
# ~/.abaper/config.yaml
base_url: https://api.bluefunda.com
realm: trm
org: default
```

### Environment Variables

| Variable          | Description                |
|-------------------|----------------------------|
| `ABAPER_BASE_URL` | Override the API base URL  |
| `ABAPER_REALM`    | Override the Keycloak realm|
| `ABAPER_ORG`      | Override the organization  |

## Authentication

ABAPer CLI uses the **OAuth2 device authorization flow** via Keycloak:

1. `abaper login` requests a device code from the authorization server
2. Your browser opens to the verification URL
3. The CLI polls for authorization completion
4. Access and refresh tokens are stored locally

Tokens are **automatically refreshed** when expired.

## SDK Usage

This repo also exports a reusable Go SDK for SAP ABAP development tooling.

### ADT Client (direct SAP connection)

```go
import (
    "github.com/bluefunda/abaper/types"
    "github.com/bluefunda/abaper/internal/adt"
)

cfg := types.ADTConfig{
    Host:     "my-sap.example.com",
    Client:   "100",
    Username: "user",
    Password: "pass",
}
client := adt.NewADTClient(cfg)
src, err := client.GetProgram("ZMY_PROGRAM")
```

### LSP Server

```go
import "github.com/bluefunda/abaper/lsp"

srv := lsp.NewServer(lsp.Config{
    ActivateOnSave: true,
})
srv.RunStdio()
```

## Man Page

A Unix man page is included with release archives.

```bash
sudo make install-man
man abaper
```

## Docker Usage

```bash
docker pull bluefunda/abaper:latest
docker run --rm -v ~/.abaper:/root/.abaper bluefunda/abaper status
```

## Developer Setup

### Prerequisites

- Go 1.25.1+
- golangci-lint (for linting)

### Build

```bash
make build          # Build for current platform
make build-all      # Cross-compile for all platforms
make test           # Run tests
make lint           # Run linter
make vet            # Run go vet
```

### Docker

```bash
make docker-build   # Build Docker image
make docker-run     # Run CLI in container
```

## Architecture

```
cmd/abaper/         # CLI entry point
internal/
  commands/         # Cobra command definitions
  client/           # ABAPer gateway HTTP client + OAuth2
  config/           # Viper-based config management
  adt/              # ADT HTTP client (direct SAP)
  lsp/              # LSP server internals
types/              # Shared ADT types and interfaces
lsp/                # Public LSP server wrapper
rest/               # REST API server
lib/                # Library wrapper for embedding
pkg/output/         # Output formatting utilities
```

- **Authentication**: OAuth2 device authorization flow via Keycloak
- **API Client**: Calls ABAPer APIs through the KrakenD gateway (`abaper-gw`) at `/abaper/api/v1/*`
- **SDK**: Types and ADT client for direct SAP ADT REST API usage

## Release Process

Releases are automated via [release-please](https://github.com/googleapis/release-please):

1. Merge PRs with [conventional commit](https://www.conventionalcommits.org/) titles
2. release-please creates a Release PR with version bump and changelog
3. Merging the Release PR triggers:
   - GoReleaser builds multi-platform binaries (named `abaper_<version>_<os>_<arch>`)
   - Binaries attached to GitHub Release
   - Docker image pushed to `bluefunda/abaper`

## License

Licensed under the [Apache License 2.0](LICENSE).

Copyright 2025 BlueFunda, Inc.
