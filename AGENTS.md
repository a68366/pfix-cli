pfix is an unofficial, public, open-source command-line client for the Planfix REST API, written in Go. It ships as a single self-contained binary. It is an independent project — not affiliated with, endorsed, sponsored, or funded by Planfix.

## Status

Milestones 1–16 are implemented and merged to `main`:
- **M1:** the config/profile layer, the Planfix transport client, `auth` (login/status/logout), and the raw `api` passthrough.
- **M2:** the typed `task` command group (`list`, `view`, `create`, `update`, `comment list`, `comment add`) and the `internal/output` rendering layer (table/detail/raw-JSON) that makes `--json`/`--fields`/`--quiet` meaningful.
- **M3:** the typed `project` command group (`list`, `view`, `create`, `update` — projects have no comments), plus extraction of the shared command helpers into `cmdutil` (`FieldsCSV`/`ValidateID`/`DecodeJSON`/`ClientFunc`) and `output` (`ColumnsFor`) so every resource reuses them.
- **M4:** the typed `contact` command group (`list`, `view`, `create`, `update`) for people and companies. `contact create` requires `--template` (Planfix rejects a templateless contact).
- **M5:** the typed `user` command group (`list`, `view` — read-only; the API disables user create and update is sensitive). Resolves the `owner`/`assignee` `user:N` references on tasks/projects.
- **M6:** the typed `report` command group (`list`, `view` — read-only). `view` decodes the single-report response defensively: Planfix returns it under the misspelled key `repost`, with a `report` fallback.
- **M7:** the `config` command group (`list`, `use`, `show`) for managing profiles locally (no API), plus `cmdutil.MaskToken` shared with `auth status`.
- **M8:** the typed `datatag` command group (`list`, `view` — read-only). Envelope keys are camelCase (`dataTags`/`dataTag`).
- **M9:** input-validation hardening — `cmdutil.ValidateID` now rejects non-positive ids across every resource.
- **M10:** the typed `template` command (`list <type>` — read-only). A new GET-based shape: `GET /<type>/templates` with an object-type path segment and no pagination; adds shared `cmdutil.ValidateObjectType`.
- **M11:** the typed `customfield` command (`list <type>` — read-only). `GET /customfield/<type>` (fixed prefix + type segment), envelope `customfields`; columns ID/NAME/TYPE.
- **M12:** the typed `object` command group (`list`, `view` — read-only). POST-list + GET-view with pagination; envelopes `objects`/`object`; object status is **fat** so STATUS uses `status.name`.
- **M13:** a `--filter <json>` pass-through on the 7 POST-list commands (task/project/contact/user/report/datatag/object) via `cmdutil.ApplyFilter` — forwards a raw Planfix `filters` array in the request body. GET-based `template`/`customfield` are excluded.
- **M14:** typed field flags on `task create`/`task update` — `--template` (create-only), `--project`, `--parent`, `--status` (now also on create), `--priority` (client-validated: the API silently resets invalid values to `NotUrgent`), `--counterparty` (contact id or `contact:N`), `--assignees`/`--auditors`/`--participants` (comma-separated `user:N`/`contact:N`/`group:N`; update replaces the list), `--start-date`/`--end-date` (ISO input → Planfix `dd-MM-yyyy`/`HH:mm`, interpreted in the account timezone). Shared parsers `cmdutil.ParsePeople`/`cmdutil.ParseTimePoint`; task-local `taskFields` registers/applies the flag set for both commands.
- **M15:** the `ping` command (`GET /ping`) — a connectivity + token-validity check that prints `OK` (`--json` passes the raw `{"result":"success"}` through; `-q` prints nothing and just sets the exit code). `auth status` now validates the token via `GET /ping` instead of `POST /task/list` — lighter and scope-independent (a task-list probe would misreport a valid token scoped only to, say, contacts). Adds shared `cmdutil.DescribeAPIError`, which maps the Planfix auth app-codes to actionable hints (code 1 unknown token → `pfix auth login`; code 5 scope denied → the token lacks the scope), used by both `ping` and `auth status`.
- **M16:** saved task filters — the read-only `task filters` command (`POST /task/filters`; envelope `filters`; columns ID/NAME/OWNER via `owner.name`) and a `--saved-filter <id>` flag on `task list` that forwards a `filterId` in the request body. The id is an opaque string (system tokens `:all`/`:in`/`:out`/`:audit` or a numeric id) forwarded verbatim; an unknown id surfaces the API's `code 41` error. This is distinct from the declined *typed filter flags* — it applies an existing named filter rather than building a `filters` array. When both are supplied, `--saved-filter` and `--filter` combine as a logical AND — the raw filter further narrows the saved view (both constraints apply; verified live against the API).
- **M17:** the global `--jq <expr>` flag — filters JSON output through an embedded jq engine (`github.com/itchyny/gojq`, no external `jq` binary needed). Setting `--jq` implies `--json`, and the expression is compiled and validated up front (`GlobalOpts.PreRun`), so an invalid expression fails before any API call. Applied at the single JSON choke point, `output.EmitJSON`, which every command now calls instead of `output.JSON` directly — so it works wherever JSON is emitted: all typed commands, `ping`, and `api`.

All tested. Still to come: `directory`/`file` as access allows. `process` is not exposed via REST (postponed); deletes, `user update`, typed filter flags, and color were declined by the user. Keep this file in sync as code lands.

## Project rules

- **Public and vendor-neutral.** Describe pfix's behavior on its own terms. Committed artifacts (code, comments, docstrings, identifiers, fixtures, docs, README, commit messages) must not name, reference, or compare against other products or tools, and must not include copied or cited third-party material.
- **Unofficial.** pfix is an independent project with no affiliation to Planfix. Keep the disclaimers in README.md and this file accurate.
- **Public dependencies only.** Every dependency must be installable from public sources. No private package indexes or internal libraries.
- **License:** Apache-2.0.

## Stack

- Go (latest stable) with `github.com/spf13/cobra` for the command tree.
- Standard-library `net/http` for the API client; `gopkg.in/yaml.v3` for config; `golang.org/x/time/rate` for request throttling; `golang.org/x/term` for hidden token entry; `github.com/itchyny/gojq` — embedded jq engine for `--jq` output filtering.
- Module path: `github.com/a68366/pfix-cli`. Binary: `pfix`.
- Deliberately lean: no config framework (no Viper), no color libraries; `gojq` is the one dependency added for a specific feature (`--jq`), kept direct and minimal (its `timefmt-go` transitive stays `// indirect`).

## Layout

Implemented:
- `main.go` — thin entry point (`cmd.Execute`).
- `internal/cmd/` — Cobra commands: `root`, `version`, `ping` (connectivity + token check), `auth/` (login/status/logout), `api/`, `task/` (`list`, `view`, `create`, `update`, and the `comment` sub-group), `project/` (`list`, `view`, `create`, `update`), `contact/` (`list`, `view`, `create`, `update`), `user/` (`list`, `view` — read-only), `report/` (`list`, `view` — read-only), `datatag/` (`list`, `view` — read-only), `template/` (`list <type>` — read-only, GET-based), `customfield/` (`list <type>` — read-only, GET-based), `object/` (`list`, `view` — read-only), `config/` (`list`, `use`, `show` — local profile management). The data package `internal/config` is imported as `pfconfig` inside `internal/cmd/config` to avoid the package-name collision.
- `internal/cmdutil/` — `GlobalOpts` (persistent flags), the `Client()`/`ClientFunc()` helpers that build a configured client from the active profile, and the resource-agnostic command helpers shared by every typed command (`FieldsCSV`, `ValidateID`, `DecodeJSON`, `ApplyFilter`, `ParsePeople`, `ParseTimePoint`, `DescribeAPIError`).
- `internal/planfix/` — Planfix REST client. A low-level `Client.Do(ctx, method, path, body, headers)` carries auth, throttling, and retries; `Client.JSON(ctx, method, path, body)` is the typed-command convenience over it (marshals the body, returns raw response bytes, maps status ≥300 to `*APIError`). `errors.go` holds `APIError` (incl. the Planfix app `Code`)/`ParseError`.
- `internal/output/` — renders decoded JSON: `Table`/`Detail` via `text/tabwriter`, a dot-path `Flatten` (e.g. `status.name`; an object with no `name` falls back to its `id`), `ColumnsFor` (default vs `--fields`-derived columns), rune-safe `Truncate`, `JSON` (pretty-print/raw passthrough — shared with `api`), and `jq.go` (`CompileJQ`/`EmitJSON`). `EmitJSON` is the flag-aware JSON entry point every command calls now — it runs the compiled `--jq` query over the decoded response when one is set, and otherwise falls back to `JSON` unchanged.
- `internal/config/` — profile load/save (atomic, mode 0600) and value precedence (`Resolve`, `ResolveProfileName`).
- `internal/buildinfo/` — version/commit/date injected at build time.

Planned (not yet present):
- Further typed resources as needed (`datatag`, `directory`, `file`).

## Build order

1. **Done (M1):** `auth` + generic `api` — credentials/profiles plus the raw passthrough make every endpoint reachable immediately.
2. **Done (M2):** `task` — list, view, create, update, and comments + the `internal/output` rendering layer.
3. **Done (M3):** `project` — list, view, create, update + shared command-helper extraction.
4. **Done (M4):** `contact` — list, view, create, update (people + companies).
5. **Done (M5):** `user` — list, view (read-only).
6. **Done (M6):** `report` — list, view (read-only).
7. **Done (M7):** `config` — list, use, show (local profile management).
8. **Done (M8):** `datatag` — list, view (read-only).
9. **Done (M9):** input-validation hardening (`ValidateID` rejects non-positive ids).
10. **Done (M10):** `template` — list per object type (read-only, GET-based).
11. **Done (M11):** `customfield` — list per object type (read-only, GET-based).
12. **Done (M12):** `object` — list, view (read-only).
13. **Done (M13):** `--filter` JSON pass-through on the POST-list commands.
14. **Done (M14):** typed field flags on `task create`/`task update`.
15. **Done (M15):** `ping` command + `auth status` token check via `GET /ping` (shared `cmdutil.DescribeAPIError` auth-error hints).
16. **Done (M16):** saved task filters — `task filters` (read-only) + `task list --saved-filter` (`filterId` pass-through).
17. **Done (M17):** global `--jq <expr>` output filter (embedded jq engine, implies `--json`, validated up front).
18. **Next:** `directory`/`file` as access allows. `process` postponed (not in REST).

## Conventions

- Auth: Bearer token + account domain; base URL `https://<domain>/rest/...`.
- Config file: `~/.config/pfix/config.yml` (mode 0600) with multiple named profiles.
- Precedence: command-line flags > environment (`PFIX_DOMAIN`, `PFIX_TOKEN`, `PFIX_PROFILE`, `PFIX_CONFIG`) > config file. Profile name resolves through `config.ResolveProfileName` (`flag > PFIX_PROFILE > current_profile > "default"`) — use it everywhere a command needs the active profile, so the commands stay consistent.
- Output: typed `task` commands default to a human-readable table (list) or key/value detail (single object), rendered by `internal/output` (stdlib `text/tabwriter`, no color). `--json` emits the API response unmodified (pretty-printed); `--fields` overrides the requested fields and table columns; `-q/--quiet` drops the header row (lists) or prints only the affected id (create/update/comment add). `--jq <expr>` filters the JSON output through a jq expression (implies `--json`). `api` always emits raw JSON. Errors go to stderr with a non-zero exit code. The Planfix layer stays thin — commands render generically from decoded `map[string]any` via dot-paths rather than typed structs, so unconfirmed nested shapes need no model.
- Transport: `Client.Do` returns the HTTP response for any status (callers inspect `StatusCode` and use `planfix.ParseError` for detail). It retries connection errors + 5xx, never 4xx. Every request carries a `User-Agent` of `pfix/<version>` (from `buildinfo.Version`, set on the `Client.UserAgent` field in `New`); a caller-supplied `User-Agent` header — e.g. `api -H "User-Agent: ..."` — overrides it.
- API specifics: list endpoints are POST with `pageSize`/`offset`/`fields`/`filters`; fields must be requested explicitly — ship sensible per-resource defaults, overridable with `--fields`.

## Toolchain

- Use the project's Go toolchain (latest stable). The `go` directive is pinned in `go.mod`; keep the module graph tidy (`go mod tidy`) — every imported dependency must be in the direct `require` block, not `// indirect`.
- Format: `gofmt -l .` (output must be empty).
- Vet: `go vet ./...`.
- Test: `go test ./...`. Table-driven tests; stand up a fake API with `net/http/httptest`; mock only at the HTTP boundary, never the code under test.
- Lint (optional, if installed): `golangci-lint run`.
- Build: `go build -o pfix .`; release builds embed metadata via `-ldflags "-X github.com/a68366/pfix-cli/internal/buildinfo.Version=..."` (and `Commit`/`Date`).
- CI: GitHub Actions — `.github/workflows/ci.yml` runs gofmt/vet/build/`go test -race`/tidy-check plus golangci-lint (config in `.golangci.yml`) on pushes to `main` and PRs; `.github/workflows/release.yml` + `.goreleaser.yml` publish multi-platform binaries to GitHub Releases on `v*` tags via GoReleaser.
- Releasing: on a green `main`, create and push an annotated tag — `git tag -a vX.Y.Z -m "pfix vX.Y.Z" && git push origin vX.Y.Z`. The tag triggers the release workflow: GoReleaser builds linux/darwin/windows × amd64/arm64 archives (version/commit/date embedded via ldflags), writes `checksums.txt`, assembles the changelog from Conventional Commit subjects (`docs`/`test`/`chore`/`ci` types are excluded — pick commit types with the changelog in mind), and publishes the GitHub Release immediately (`draft: false` in `.goreleaser.yml`). Verify the Actions run and the Releases page; `go install github.com/a68366/pfix-cli@latest` resolves the new tag once the Go module proxy refreshes.

## Testing rules

- All behaviour changes must be covered by tests — if it isn't tested, it isn't done.
- Test decisions and branches (error mapping, config precedence, retry behavior, request building), not glue code.
