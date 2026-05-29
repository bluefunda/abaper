# ABAPer

[![Go Reference](https://pkg.go.dev/badge/github.com/bluefunda/abaper.svg)](https://pkg.go.dev/github.com/bluefunda/abaper)
[![CI](https://github.com/bluefunda/abaper/actions/workflows/ci.yml/badge.svg)](https://github.com/bluefunda/abaper/actions/workflows/ci.yml)
[![License](https://img.shields.io/badge/license-Apache%202.0-blue.svg)](LICENSE)

**CLI, TUI, and Go SDK** for the [ABAPer](https://abaper.bluefunda.com) platform — develop, deploy, and test ABAP objects from your terminal, automate SAP workflows in Go, or embed the LSP server in any editor.

## The ABAPer Ecosystem

ABAPer is available across multiple surfaces. All share the same backend and AI capabilities.

| Client | Repo | Best for |
|--------|------|----------|
| **Web IDE** | [abaper-editor](https://github.com/bluefunda/abaper-editor) | Browser-based editing, no install |
| **VS Code** | [abaper-vscode](https://github.com/bluefunda/abaper-vscode) | VS Code users |
| **CLI / TUI** | **this repo** | Terminal workflows, scripting, CI/CD |
| **Go SDK** | **this repo** (`lib/`) | Embedding in Go programs |
| **Eclipse** | [abaper-eclipse](https://github.com/bluefunda/abaper-eclipse) | Eclipse / ADT users _(coming soon)_ |

### Feature matrix

| Feature | Web IDE | VS Code | CLI / TUI |
|---------|:-------:|:-------:|:---------:|
| AI pair-programmer chat | ✓ | ✓ | ✓ |
| Generate ABAP | ✓ | ✓ | ✓ |
| Deploy & activate | ✓ | ✓ | ✓ |
| Syntax check / LSP | ✓ | ✓ | SDK |
| Run unit tests | ✓ | ✓ | ✓ |
| SAP system management | ✓ | ✓ | ✓ |
| Offline / no browser | — | — | ✓ |
| CI/CD integration | — | — | ✓ |
| Embed in Go programs | — | — | ✓ |

---

## Installation

### One-line installer (macOS / Linux)

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

### Docker

```bash
docker pull ghcr.io/bluefunda/abaper
docker run --rm -v ~/.abaper:/root/.abaper ghcr.io/bluefunda/abaper status
```

### From source

```bash
go install github.com/bluefunda/abaper/cmd/abaper@latest
```

### From GitHub Releases

Download the binary for your platform from the [releases page](https://github.com/bluefunda/abaper/releases).

| Platform        | Archive                               |
|-----------------|---------------------------------------|
| macOS (ARM64)   | `abaper_<version>_darwin_arm64.zip`   |
| macOS (AMD64)   | `abaper_<version>_darwin_amd64.zip`   |
| Linux (AMD64)   | `abaper_<version>_linux_amd64.tar.gz` |
| Linux (ARM64)   | `abaper_<version>_linux_arm64.tar.gz` |
| Windows (AMD64) | `abaper_<version>_windows_amd64.zip`  |
| Windows (ARM64) | `abaper_<version>_windows_arm64.zip`  |

---

## Quick Start

```bash
# 1. Authenticate with the ABAPer platform
abaper login

# 2. Connect to your SAP system
abaper system add --host https://my-sap:44300 -u DEVELOPER -p secret
abaper system test

# 3. Open the interactive AI chat (TUI)
abaper

# 4. Or drive commands directly
abaper generate --type program --name ZMY_REPORT
abaper deploy --type program --name ZMY_REPORT --source-file report.abap
```

---

## CLI Reference

### Authentication

```bash
abaper login       # OAuth2 device flow — opens browser
abaper logout      # Clear stored tokens
abaper status      # Show auth + API health
```

### SAP System Management

Connect the CLI and AI chat to your SAP system. Credentials are stored at `~/.abaper/systems.json` (0600) and sent as `X-SAP-*` headers on every request so the gateway can initialise the SAP MCP server for AI tools.

```bash
abaper system add --host https://my-sap:44300 -u USER -p PASS
abaper system add --host https://my-sap:44300 --name "DEV" --client 200 -u USER -p PASS
abaper system list            # show all systems (● = active)
abaper system use "DEV"       # switch active system
abaper system test            # test active system connection
abaper system test "DEV"      # test a specific system
abaper system remove "DEV"
```

You can also manage systems inside the TUI by typing `/system add`, `/system list`, or `/system edit <name>`.

### AI Pair-Programmer

```bash
abaper                        # interactive TUI (Bubble Tea chat)
abaper ai chat "Explain SELECT FOR ALL ENTRIES"
abaper ai chat "Optimise this" --context-file program.abap
abaper ai chat "What about performance?" --chat-id <id>
```

### Object Operations

```bash
# Generate (create on SAP)
abaper generate --type program --name ZMY_REPORT
abaper generate --type class --name ZCL_MY_CLASS --source-file my_class.abap
abaper generate --type interface --name ZIF_MY_INTERFACE

# Deploy (upload + activate)
abaper deploy --type program --name ZMY_REPORT --source-file report.abap

# Test
abaper test --type class --name ZCL_MY_CLASS

# List
abaper list objects --package ZDEV
abaper list packages --name ZDEV
```

**Object types:** `program`, `class`, `interface`

### Global Flags

| Flag             | Default                     | Description                   |
|------------------|-----------------------------|-------------------------------|
| `--base-url`     | `https://api.bluefunda.com` | ABAPer API gateway URL        |
| `--realm`        | `trm`                       | Keycloak realm                |
| `-o`, `--output` | `text`                      | Output format: `text`, `json` |

### Configuration

Priority: CLI flags → environment variables → `~/.abaper/config.yaml`

```yaml
# ~/.abaper/config.yaml
base_url: https://api.bluefunda.com
realm: trm
org: default
```

| Environment variable | Description             |
|----------------------|-------------------------|
| `ABAPER_BASE_URL`    | Override API base URL   |
| `ABAPER_REALM`       | Override Keycloak realm |
| `ABAPER_ORG`         | Override organisation   |

---

## Go SDK

Import the SDK to drive SAP ADT operations directly from Go — no ABAPer platform account required.

```bash
go get github.com/bluefunda/abaper/lib
```

### Connect to SAP

```go
import "github.com/bluefunda/abaper/lib"

client, err := lib.CreateADTClient(
    "https://my-sap.example.com:44300",
    "100",        // SAP client number
    "DEVELOPER",
    "secret",
)
```

### Read source

```go
src, err := client.GetProgram(ctx, "ZMY_PROGRAM")
fmt.Println(src.Source)

cls, err := client.GetClass(ctx, "ZCL_MY_CLASS")
```

### Write and activate

```go
err = client.CreateProgram(ctx, "ZMY_PROG", "My program", "$TMP", source)
result, err := client.ActivateObject(ctx, "PROG", "ZMY_PROG")
```

### Search

```go
results, err := client.SearchObjects(ctx, "ZMY_*", []string{"PROG", "CLAS"})
packages, err := client.ListPackages(ctx, "ZDEV*")
```

### Key interfaces (`types/adt.go`)

| Interface | Purpose |
|-----------|---------|
| `SourceReader` | Read ABAP object source |
| `SourceWriter` | Create and update objects |
| `PackageBrowser` | Search packages and objects |
| `ObjectActivator` | Activate objects, run unit tests |
| `LangFeatures` | LSP-style syntax check, completion, navigation |
| `ADTClient` | Full interface — all of the above |

### LSP Server

Embed SAP language intelligence in any LSP-capable editor:

```go
import "github.com/bluefunda/abaper/lsp"

srv := lsp.NewServer(adtClient, "/path/to/workspace")
srv.RunStdio()       // stdio transport (most editors)
// srv.RunTCP(":2087")
```

---

## Architecture

```
cmd/abaper/          CLI entry point (Cobra)
tui/                 Bubble Tea TUI — chat + SAP system form
internal/
  commands/          Cobra subcommands
  client/            ABAPer gateway HTTP client (OAuth2 + SAP header injection)
  config/            Config, token, and SAP system storage (~/.abaper/)
  adt/               Direct SAP ADT REST client
  lsp/               LSP server internals
types/               Shared interfaces and ADT data structures
lsp/                 Public LSP server wrapper
rest/                REST API server (CLI feature parity, no AI)
lib/                 Library wrapper for Go embedding
```

**Request flow:**

```
abaper CLI / TUI
      │  Bearer token + X-SAP-{Host,Client,User,Password} headers
      ▼
abaper-gw  (KrakenD gateway — authenticates, routes, starts SAP MCP server)
      │
      ▼
SAP ADT REST API    ◄── also reachable directly via lib/ (no gateway)
```

---

## Docker

```bash
docker pull ghcr.io/bluefunda/abaper:latest

# Persist credentials
docker run --rm -v ~/.abaper:/root/.abaper ghcr.io/bluefunda/abaper status

# One-off deploy
docker run --rm \
  -v ~/.abaper:/root/.abaper \
  -v $(pwd):/workspace \
  ghcr.io/bluefunda/abaper deploy \
    --type program --name ZMY --source-file /workspace/my.abap
```

---

## Developer Setup

**Prerequisites:** Go 1.25.1+, golangci-lint

```bash
make build        # build for current platform
make build-all    # cross-compile for all platforms
make test         # go test -race ./...
make lint         # golangci-lint
make vet          # go vet
```

---

## Release Process

Releases are automated via [release-please](https://github.com/googleapis/release-please):

1. Merge PRs with [conventional commit](https://www.conventionalcommits.org/) titles (`feat:`, `fix:`, `chore:` …)
2. release-please opens a Release PR with version bump + changelog
3. Merging the Release PR triggers GoReleaser → multi-platform binaries + Docker image at `ghcr.io/bluefunda/abaper`

---

## License

[Apache License 2.0](LICENSE) — Copyright 2025 BlueFunda, Inc.
