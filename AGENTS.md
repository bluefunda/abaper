# AGENTS.md — AI Coding Agent Instructions for abaper

## Project Overview

**abaper** is a unified Go CLI and SDK for ABAP development on the ABAPer platform and SAP systems.

Module: `github.com/bluefunda/abaper` | Go 1.25.1 | License: Apache 2.0

Two primary surfaces:

1. **CLI** — Cobra commands that communicate with the ABAPer platform gateway via OAuth2:
   - Commands: `login`, `logout`, `status`, `signup`, `system`, `search`, `list`, `get`, `generate`, `deploy`, `test`, `serve`, `ai`, `update`, `version`
   - Entry point: `cmd/abaper/main.go`

2. **SDK** — Reusable Go packages for SAP ABAP development tooling:
   - `types/` — `ADTClient` interface, all ABAP data structs
   - `internal/adt/` — ADT HTTP client implementation (direct SAP)
   - `internal/lsp/` — LSP server internals
   - `lsp/` — public LSP server wrapper
   - `rest/` — REST API server
   - `lib/` — library wrapper

---

## Build and Verification

```sh
make build        # Build for current platform (bin/abaper)
make build-all    # Cross-compile for all platforms
make test         # go test ./...
make lint         # golangci-lint run
make vet          # go vet ./...
```

Linter config: `.golangci.yml`.

---

## Directory Structure

```
cmd/abaper/         # CLI entry point (package main)

internal/
  commands/         # Cobra command definitions (login, generate, deploy, etc.)
  client/           # ABAPer gateway HTTP client + OAuth2 device flow
  config/           # Viper-based config (~/.abaper/config.yaml); tokens read from bai's shared store
  adt/              # ADT HTTP client (ADTClientImpl — ~50 methods, 3400 lines)
  lsp/
    abap/           # ABAP language knowledge (keywords, formatter, syntax)
    abaplint/       # abaplint CLI integration
    backend/        # LSPBackend interface + 3 implementations (Offline, ADT, Hybrid)
    document/       # Open document tracking + URI utilities
    handler/        # LSP protocol handlers (8 files)

types/              # ADTClient interface + all ADT data structs (package types)
lsp/                # Public LSP server wrapper (NewServer, RunStdio, RunTCP)
rest/               # REST API server + GitHub OAuth proxy
lib/                # Library wrapper for embedding
pkg/output/         # Output formatting utilities (JSON / table)

man/                # Unix man page
scripts/            # Build and release scripts
hurl/               # HTTP test files (SAP ADT API)
editors/zed/        # Zed editor extension (Rust)
examples/           # Docker examples
testdata/           # ABAP test fixtures
docs/               # Documentation and Hugo site
```

---

## Key Interfaces

### ADTClient (`types/adt.go`)

Central contract for SAP ABAP operations. ~50 methods:
- Retrieval: `GetProgram`, `GetClass`, `GetFunction`, `GetInclude`, `GetInterface`, `GetStructure`, `GetTable`
- Creation: `CreateProgram`, `CreateClass`, `CreateInclude`, etc.
- Update: `UpdateProgram`, `UpdateClass`, `UpdateInclude`, `UpdateInterface`
- Search/list: `SearchObjects`, `ListPackages`, `GetPackageContents`
- Connection: `TestConnection`, `IsAuthenticated`, `Authenticate`, `SetSessionType`
- LSP support: `SyntaxCheck`, `GetCompletionProposals`, `GetNavigationTarget`
- Activation/testing: `ActivateObject`, `RunUnitTests`

Single implementation: `internal/adt/client.go` (`ADTClientImpl`).

### LSPBackend (`internal/lsp/backend/interface.go`)

```go
type LSPBackend interface {
    SyntaxCheck(objectType, objectName, source string) (*types.SyntaxCheckResult, error)
    Complete(objectType, objectName, source string, line, column int) ([]types.CompletionProposal, error)
    Navigate(objectType, objectName, source string, line, column int) (*types.NavigationTarget, error)
    Format(source string) (string, error)
    Activate(objectType, objectName string) (*types.ActivationResult, error)
    IsConnected() bool
}
```

Three implementations: `OfflineBackend`, `ADTBackend`, `HybridBackend`.

---

## CLI Architecture (ABAPer Gateway)

- Config: CLI flags > env vars (`ABAPER_BASE_URL`, `ABAPER_REALM`) > `~/.abaper/config.yaml`
- `abaper login`/`abaper logout` delegate entirely to `bai login`/`bai logout`. abaper authenticates against the same Keycloak realm and OAuth client as bai (see `internal/config.ClientID`/`DefaultRealm`) and reads bai's stored credentials directly (`~/.bai/config.yaml` + OS keychain, service `bai`) — a single `bai login` is sufficient for AI features (`ai chat`/`ai code`) and every gateway command (`search`, `list`, `generate`, `deploy`, `test`, `system test`). abaper keeps no token store of its own.
- Tokens: sourced from `~/.bai/config.yaml` / OS keychain (owned by `bai`); abaper transparently refreshes and writes the refreshed pair back into that same store when expired.
- API calls: through ABAPer KrakenD gateway at `/abaper/api/v1/*`
- Response: `{ "success": bool, "data": T, "error": string }`
- Headers: `Authorization: Bearer`, `X-Realm`, `X-SAP-*`

### Known limitation

`abaper deploy` always calls the create-object API — it has no update/save path. Deploying to an object name that already exists on the SAP system fails with "A program or include already exists". It currently only works as a create+activate shortcut for brand-new objects, not as a way to push changes to an existing one.

### AI commands (`ai chat`, `ai code`, bare TUI chat)

- All three use the embedded `bluefunda-ai` SDK (`sdk/agent`), which grants a shell/bash tool by default (`agent.Options.OnToolCall` falls back to `DefaultExecute` when unset — none of `ai.go`/`ai_code.go`/`tui/chat.go` override it).
- `ai chat`/TUI chat run with `MaxTurns: 5` (a hard cap on agentic-loop iterations per call, not just per session) — enough for a couple of tool round-trips, but not a full agentic loop. `ai code` defaults to 20 (`--max-turns` overridable).
- Their system prompts explicitly tell the model to reach the SAP system via `abaper get`/`search`/`list` through the bash tool — without that hint, the model tries to `search_files`/read local disk for objects that only exist on SAP, burns its turn budget on a call that can never succeed, and dies with "Reached max-turns limit" before producing any answer. If you add a new gateway-backed lookup command, mention it in these system prompts too, or the AI commands won't know it exists.
- `abaper get` (`internal/commands/get.go`) exists specifically so these prompts have something to shell out to — it wraps the existing `client.GetObject` gateway call, which had no CLI command before.

### CLI command reference

Global flags on every command: `--base-url` (default `https://api.bluefunda.com`), `--realm` (default: see `internal/config.DefaultRealm`), `-o/--output` (`text`|`json`, default `text`).

| Command | Required | Notes |
|---|---|---|
| `abaper login` | — | delegates to `bai login` |
| `abaper logout` | — | delegates to `bai logout` |
| `abaper status` | — | reports auth + gateway reachability |
| `abaper signup` | — | opens the BlueFunda signup page |
| `abaper search <pattern>` | positional pattern | `-t/--type` to filter (e.g. `PROG/P`) |
| `abaper list objects` | — | `--package`, `--type` optional filters |
| `abaper list packages --name <pkg>` | `--name` | |
| `abaper get --name <n>` | `--name` | `--type` program\|class\|interface (default `program`); fetches existing source, used by `ai chat`/`ai code`/TUI chat for review prompts |
| `abaper generate --name <n> --source-file <f>` | `--name`, source via `--source-file` or stdin | `--type` program\|class\|interface (default `program`) |
| `abaper deploy --name <n> --source-file <f>` | `--name`, `--source-file` | `--type` program\|class\|interface (default `program`); create-only, see Known limitation |
| `abaper test --name <n>` | `--name` | `--type` class\|program (default `class`) |
| `abaper system add --host <url> -u <user> -p <pass>` | `--host`, `-u`, `-p` | `--name`, `--client` (default `100`) optional |
| `abaper system list` | — | `●` marks the active system |
| `abaper system use <name>` | positional name | |
| `abaper system test [name]` | — | tests active system if name omitted |
| `abaper system remove <name>` (aliases `rm`, `delete`) | positional name | |
| `abaper ai chat <prompt>` | positional prompt | `--model` auto\|fast\|think, `--context-file` |
| `abaper ai code <prompt>` | positional prompt (required — `[prompt]` in `--help` is misleading, at least 1 arg enforced) | `--model`, `--context-file`, `--max-turns` (default 20), `--verbose` |
| `abaper serve` | — | starts the local REST server |
| `abaper update` | — | self-update |
| `abaper version` | — | |
| `abaper` (no subcommand) | — | opens the interactive TUI |

## SDK Constraints

- ADT types use both `xml` and `json` struct tags. Maintain both when adding fields.
- Object names are uppercased before ADT calls.
- LSP `HybridBackend` tries ADT first, falls back to offline on error, runs 60-second connection monitor.
- No vendored dependencies.

---

## Environment Variables

### CLI
| Variable          | Purpose                          |
|-------------------|----------------------------------|
| `ABAPER_BASE_URL` | ABAPer API base URL              |
| `ABAPER_REALM`    | Keycloak realm                   |
| `ABAPER_ORG`      | Organization                     |

### SDK (ADT)
| Variable       | Purpose                          |
|----------------|----------------------------------|
| `SAP_HOST`     | SAP system hostname              |
| `SAP_PORT`     | SAP system port                  |
| `SAP_CLIENT`   | SAP client number                |
| `SAP_USERNAME` | SAP username                     |
| `SAP_PASSWORD` | SAP password                     |
| `GH_PAT`       | GitHub personal access token     |

---

## Commit Conventions

- Conventional format: `feat:`, `fix:`, `chore:`, `docs:`, `refactor:`, `perf:`, `security:`, `infra:`
- Scoped: `feat(lsp): add go-to-definition for includes`
- Branches: `<type>/<short-description>`
- PRs target `main`, squash-merged

## Release

Automated via release-please (conventional commits drive changelog) + GoReleaser.

---

## How we work (operating rules)

- Every task starts from a GitHub issue. Read it fully first: `gh issue view <N> 2>/dev/null`.
- M/L issues: produce a short plan and WAIT for approval before editing. S: proceed.
- Respect the issue's "Out of scope" — do not refactor or touch anything outside it.
- Commit in small steps, Conventional Commits, referencing the issue: `feat(abaper): ... (#<N>)`.
- Before claiming done: run the issue's "Verify with" commands; confirm every acceptance box.
- Open the PR with "Closes #<N>" + a one-paragraph, release-note-ready summary.
- Secrets come from Vault/env, never hardcoded. Never run destructive git/deploy without asking.
