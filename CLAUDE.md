pfix is a public, open-source command-line client for the Planfix REST API, written in Go. It ships as a single self-contained binary and serves two audiences: people working in a terminal, and automated or AI agents that consume machine-readable output.

## Status

Bootstrap phase. The Go module and command tree are still being scaffolded — the architecture below is the target, not a description of what already exists. Keep this file in sync as code lands.

## Project rules

- **Public and vendor-neutral.** Describe pfix's behavior on its own terms. Committed artifacts (code, comments, docstrings, identifiers, fixtures, docs, README, commit messages) must not name, reference, or compare against other products or tools, and must not include copied or cited third-party material.
- **Public dependencies only.** Every dependency must be installable from public sources. No private package indexes or internal libraries.
- **License:** Apache-2.0.

## Stack

- Go (latest stable) with Cobra for the command tree.
- Standard-library `net/http` for the API client; `gopkg.in/yaml.v3` for config; `golang.org/x/time/rate` for request throttling.
- Module path: `github.com/a68366/pfix-cli` (set at `go mod init`).

## Layout (target)

- `main.go` — thin entry point.
- `internal/cmd/` — Cobra commands: `root`, `auth/`, `task/` (including `comment`), `project/`, `config/`, `api/`.
- `internal/planfix/` — Planfix REST client. A single low-level `Client.Do(ctx, method, path, body, headers)` carries auth, throttling, and retries; the typed commands and the raw `api` command both go through it.
- `internal/config/` — profile loading/saving and value precedence.
- `internal/output/` — table rendering and raw-JSON emission.

## Build order (MVP)

1. `auth` + generic `api` — credentials/profiles plus the raw passthrough come first; together they make every endpoint reachable immediately.
2. `task` — list, view, create, update, and comments.
3. `project`, then the remaining resources.

## Conventions

- Auth: Bearer token + account domain; base URL `https://<domain>/rest/...`.
- Config file: `~/.config/pfix/config.yml` (mode 0600) with multiple named profiles.
- Precedence: command-line flags > environment (`PFIX_DOMAIN`, `PFIX_TOKEN`, `PFIX_PROFILE`) > config file.
- Output: human-readable table by default; `--json` emits the API response unmodified (raw passthrough). Errors go to stderr with a non-zero exit code.
- API specifics: list endpoints are POST with `pageSize`/`offset`/`fields`/`filters`; fields must be requested explicitly — ship sensible per-resource defaults, overridable with `--fields`.

## Toolchain

- Format: `gofmt -l .` (output must be empty).
- Vet: `go vet ./...`.
- Lint: `golangci-lint run`.
- Test: `go test ./...`. Table-driven tests; stand up a fake API with `net/http/httptest`; mock only at the HTTP boundary, never the code under test.
- Build: `go build ./...`; release builds embed version metadata via `-ldflags -X`.

## Testing rules

- All behaviour changes must be covered by tests — if it isn't tested, it isn't done.
- Test decisions and branches (error mapping, config precedence, pagination, request building), not glue code.
