pfix is a public, open-source command-line client for the Planfix REST API, written in Go. It ships as a single self-contained binary and serves two audiences: people working in a terminal, and automated or AI agents that consume machine-readable output.

## Status

Milestone 1 is implemented and merged to `main`: the config/profile layer, the Planfix transport client, `auth` (login/status/logout), and the raw `api` passthrough — all tested. Still to come: the typed resource commands (`task`, then `project`/`comment`) and the human-readable table output layer. Keep this file in sync as code lands.

## Project rules

- **Public and vendor-neutral.** Describe pfix's behavior on its own terms. Committed artifacts (code, comments, docstrings, identifiers, fixtures, docs, README, commit messages) must not name, reference, or compare against other products or tools, and must not include copied or cited third-party material.
- **Public dependencies only.** Every dependency must be installable from public sources. No private package indexes or internal libraries.
- **License:** Apache-2.0.

## Stack

- Go (latest stable) with `github.com/spf13/cobra` for the command tree.
- Standard-library `net/http` for the API client; `gopkg.in/yaml.v3` for config; `golang.org/x/time/rate` for request throttling; `golang.org/x/term` for hidden token entry.
- Module path: `github.com/a68366/pfix-cli`. Binary: `pfix`.
- Deliberately lean: no config framework (no Viper), no color libraries.

## Layout

Implemented:
- `main.go` — thin entry point (`cmd.Execute`).
- `internal/cmd/` — Cobra commands: `root`, `version`, `auth/` (login/status/logout), `api/`.
- `internal/cmdutil/` — `GlobalOpts` (persistent flags) and the `Client()` helper that builds a configured client from the active profile.
- `internal/planfix/` — Planfix REST client. A single low-level `Client.Do(ctx, method, path, body, headers)` carries auth, throttling, and retries; the raw `api` command and the future typed commands all go through it. `errors.go` holds `APIError`/`ParseError`.
- `internal/config/` — profile load/save (atomic, mode 0600) and value precedence (`Resolve`, `ResolveProfileName`).
- `internal/buildinfo/` — version/commit/date injected at build time.

Planned (not yet present):
- `internal/cmd/task/` (+ `comment`), `internal/cmd/project/`, `internal/cmd/config/`.
- `internal/output/` — table rendering; the `--json` flag becomes meaningful once typed commands have a non-JSON default.

## Build order

1. **Done (M1):** `auth` + generic `api` — credentials/profiles plus the raw passthrough make every endpoint reachable immediately.
2. **Next (M2):** `task` — list, view, create, update, and comments.
3. `project`, then the remaining resources.

## Conventions

- Auth: Bearer token + account domain; base URL `https://<domain>/rest/...`.
- Config file: `~/.config/pfix/config.yml` (mode 0600) with multiple named profiles.
- Precedence: command-line flags > environment (`PFIX_DOMAIN`, `PFIX_TOKEN`, `PFIX_PROFILE`, `PFIX_CONFIG`) > config file. Profile name resolves through `config.ResolveProfileName` (`flag > PFIX_PROFILE > current_profile > "default"`) — use it everywhere a command needs the active profile, so the commands stay consistent.
- Output: `--json` will emit the API response unmodified (raw passthrough); typed commands will default to a human-readable table. Today `api` always emits raw JSON, pretty-printed via stdlib `json.Indent` (no color). Errors go to stderr with a non-zero exit code.
- Transport: `Client.Do` returns the HTTP response for any status (callers inspect `StatusCode` and use `planfix.ParseError` for detail). It retries connection errors + 5xx, never 4xx.
- API specifics: list endpoints are POST with `pageSize`/`offset`/`fields`/`filters`; fields must be requested explicitly — ship sensible per-resource defaults, overridable with `--fields`.

## Toolchain

- Use the project's Go toolchain (latest stable). The `go` directive is pinned in `go.mod`; keep the module graph tidy (`go mod tidy`) — every imported dependency must be in the direct `require` block, not `// indirect`.
- Format: `gofmt -l .` (output must be empty).
- Vet: `go vet ./...`.
- Test: `go test ./...`. Table-driven tests; stand up a fake API with `net/http/httptest`; mock only at the HTTP boundary, never the code under test.
- Lint (optional, if installed): `golangci-lint run`.
- Build: `go build -o pfix .`; release builds embed metadata via `-ldflags "-X github.com/a68366/pfix-cli/internal/buildinfo.Version=..."` (and `Commit`/`Date`).

## Testing rules

- All behaviour changes must be covered by tests — if it isn't tested, it isn't done.
- Test decisions and branches (error mapping, config precedence, retry behavior, request building), not glue code.
