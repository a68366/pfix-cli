pfix is a public, open-source command-line client for the Planfix REST API, written in Go. It ships as a single self-contained binary and serves two audiences: people working in a terminal, and automated or AI agents that consume machine-readable output.

## Status

Milestones 1–4 are implemented and merged to `main`:
- **M1:** the config/profile layer, the Planfix transport client, `auth` (login/status/logout), and the raw `api` passthrough.
- **M2:** the typed `task` command group (`list`, `view`, `create`, `update`, `comment list`, `comment add`) and the `internal/output` rendering layer (table/detail/raw-JSON) that makes `--json`/`--fields`/`--quiet` meaningful.
- **M3:** the typed `project` command group (`list`, `view`, `create`, `update` — projects have no comments), plus extraction of the shared command helpers into `cmdutil` (`FieldsCSV`/`ValidateID`/`DecodeJSON`/`ClientFunc`) and `output` (`ColumnsFor`) so every resource reuses them.
- **M4:** the typed `contact` command group (`list`, `view`, `create`, `update`) for people and companies. `contact create` requires `--template` (Planfix rejects a templateless contact).

All tested. Still to come: the remaining Planfix resources. Keep this file in sync as code lands.

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
- `internal/cmd/` — Cobra commands: `root`, `version`, `auth/` (login/status/logout), `api/`, `task/` (`list`, `view`, `create`, `update`, and the `comment` sub-group), `project/` (`list`, `view`, `create`, `update`), `contact/` (`list`, `view`, `create`, `update`).
- `internal/cmdutil/` — `GlobalOpts` (persistent flags), the `Client()`/`ClientFunc()` helpers that build a configured client from the active profile, and the resource-agnostic command helpers shared by every typed command (`FieldsCSV`, `ValidateID`, `DecodeJSON`).
- `internal/planfix/` — Planfix REST client. A low-level `Client.Do(ctx, method, path, body, headers)` carries auth, throttling, and retries; `Client.JSON(ctx, method, path, body)` is the typed-command convenience over it (marshals the body, returns raw response bytes, maps status ≥300 to `*APIError`). `errors.go` holds `APIError` (incl. the Planfix app `Code`)/`ParseError`.
- `internal/output/` — renders decoded JSON: `Table`/`Detail` via `text/tabwriter`, a dot-path `Flatten` (e.g. `status.name`; an object with no `name` falls back to its `id`), `ColumnsFor` (default vs `--fields`-derived columns), rune-safe `Truncate`, and `JSON` (pretty-print/raw passthrough — shared with `api`).
- `internal/config/` — profile load/save (atomic, mode 0600) and value precedence (`Resolve`, `ResolveProfileName`).
- `internal/buildinfo/` — version/commit/date injected at build time.

Planned (not yet present):
- `internal/cmd/config/`, and further typed resources.

## Build order

1. **Done (M1):** `auth` + generic `api` — credentials/profiles plus the raw passthrough make every endpoint reachable immediately.
2. **Done (M2):** `task` — list, view, create, update, and comments + the `internal/output` rendering layer.
3. **Done (M3):** `project` — list, view, create, update + shared command-helper extraction.
4. **Done (M4):** `contact` — list, view, create, update (people + companies).
5. **Next (M5):** the remaining Planfix resources.

## Conventions

- Auth: Bearer token + account domain; base URL `https://<domain>/rest/...`.
- Config file: `~/.config/pfix/config.yml` (mode 0600) with multiple named profiles.
- Precedence: command-line flags > environment (`PFIX_DOMAIN`, `PFIX_TOKEN`, `PFIX_PROFILE`, `PFIX_CONFIG`) > config file. Profile name resolves through `config.ResolveProfileName` (`flag > PFIX_PROFILE > current_profile > "default"`) — use it everywhere a command needs the active profile, so the commands stay consistent.
- Output: typed `task` commands default to a human-readable table (list) or key/value detail (single object), rendered by `internal/output` (stdlib `text/tabwriter`, no color). `--json` emits the API response unmodified (pretty-printed); `--fields` overrides the requested fields and table columns; `-q/--quiet` drops the header row (lists) or prints only the affected id (create/update/comment add). `api` always emits raw JSON. Errors go to stderr with a non-zero exit code. The Planfix layer stays thin — commands render generically from decoded `map[string]any` via dot-paths rather than typed structs, so unconfirmed nested shapes need no model.
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
