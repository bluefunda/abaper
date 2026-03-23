# CLAUDE.md — abaper-cli

## What is this?

CLI client for the ABAPer platform. Communicates with ABAPer APIs through the KrakenD gateway (`abaper-gw`).

Module: `github.com/bluefunda/abaper-cli` | Go 1.25

## Build & Verify

```bash
make build        # Build for current platform
make build-all    # Cross-compile for all platforms
make test         # Run tests
make lint         # Run linter
make vet          # Run go vet
```

## Commands

`login`, `logout`, `status`, `generate`, `deploy`, `test`, `list objects`, `list packages`, `ai chat`, `version`

## Key Patterns

- OAuth2 device authorization flow via Keycloak (`cai-cli` client ID)
- Config: CLI flags > env vars (`ABAPER_BASE_URL`, `ABAPER_REALM`) > `~/.abaper/config.yaml`
- Tokens stored at `~/.abaper/tokens.yaml` (0600 perms), auto-refreshed
- API calls go through gateway at `/abaper/api/v1/*`
- Response format: `{ "success": bool, "data": T, "error": string }`
- Headers: `Authorization: Bearer`, `X-Realm`, `X-SAP-*`

## Conventions

- Commits: conventional format (`feat:`, `fix:`, `chore:`)
- Branches: `<type>/<short-description>`
- PRs: conventional commit title, target `main`, squash-merged
- Releases: release-please + GoReleaser for multi-platform binaries + Docker image
