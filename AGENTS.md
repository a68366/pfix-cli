pfix is an unofficial, public, open-source command-line client for the Planfix REST API, written in Go. It ships as a single self-contained binary. It is an independent project — not affiliated with, endorsed, sponsored, or funded by Planfix.

This file holds only the rules and the API behavior that change how code must be written — not a history. What shipped when is in the GitHub release notes (built from `feat`/`fix` commit subjects); when a change uncovers new API behavior that future code must respect, add it to **API behavior to design around** below.

## Project rules

- **Public and vendor-neutral.** Describe pfix's behavior on its own terms. Committed artifacts (code, comments, docstrings, identifiers, fixtures, docs, README, commit messages) must not name, reference, or compare against other products or tools, and must not include copied or cited third-party material. (Sole exception: a committed `CLAUDE.md` whose entire content is the import line `@AGENTS.md`, so agent tooling that looks for that filename loads this vendor-neutral file instead. `AGENTS.md` is the canonical, tool-agnostic source; `CLAUDE.md` is only a pointer to it.)
- **Unofficial.** pfix is an independent project with no affiliation to Planfix. Keep the disclaimers in README.md and this file accurate.
- **Public dependencies only.** Every dependency must be installable from public sources. No private package indexes or internal libraries.
- **License:** Apache-2.0.

## Stack

- Go (latest stable) with `github.com/spf13/cobra` for the command tree.
- Standard-library `net/http` for the API client; `gopkg.in/yaml.v3` for config; `golang.org/x/time/rate` for request throttling; `golang.org/x/term` for hidden token entry; `github.com/itchyny/gojq` — embedded jq engine for `--jq`.
- Module path: `github.com/a68366/pfix-cli`. Binary: `pfix`.
- Deliberately lean: no config framework, no color libraries. A new dependency needs a specific feature that the standard library cannot reasonably cover, and stays in the direct `require` block.

## Layout

- `cmd/pfix/main.go` — canonical entry point (`go install .../cmd/pfix@latest` yields `pfix`). The root `main.go` is a compatibility shim so the module-root `go install` keeps working (as `pfix-cli`). Both only call `cmd.Main()` (`internal/cmd/main.go`), which wires `signal.NotifyContext(os.Interrupt)` so `Ctrl-C` cancels long-running commands (downloads, uploads).
- `internal/cmd/<resource>/` — one Cobra package per command group (`task`, `project`, `contact`, `user`, `report`, `datatag`, `template`, `customfield`, `object`, `file`, `comment`, `config`, `auth`, `api`), plus `whoami` (also used by `auth status`), and `root`, `version`, `ping`. `internal/config` is imported as `pfconfig` inside `internal/cmd/config` to avoid the name collision.
- `internal/cmd/{processes,groups,files}/` — list commands shared by several parents (e.g. `files.NewCmd(g, files.Options{...})` backs `task|contact|project files`, each registering only the flags its endpoint supports).
- `internal/cmdutil/` — `GlobalOpts` (persistent flags), `Client()`/`ClientFunc()`, and the resource-agnostic helpers every command reuses (`ValidateID`, `FieldsCSV`, `DecodeJSON`, `ApplyFilter`, `ParsePeople`, `ParseTimePoint`, `DescribeAPIError`, `ParseCustomFieldSpecs`/`BuildCustomFieldData`, `FieldsHelp`, `ScanFileIDs`, `SafeFileName`, `CheckResult`). Check here before writing a helper.
- `internal/planfix/` — the REST client: `Do` (auth, throttle, retries), `JSON` (typed-command convenience; status ≥300 → `*APIError` with the app `Code`), `JSONOnce` (single attempt), `Stream` (GET with the body unread, no whole-request timeout), `Upload` (multipart, single attempt, streamed).
- `internal/output/` — `Table`/`Detail` (tabwriter), dot-path `Flatten`, `ColumnsFor`, optional `Column.Format` hook (`Table` only), `EmitJSON` (the `--jq`-aware JSON entry point).
- `internal/config/` — profile load/save (atomic, 0600) and precedence (`Resolve`, `ResolveProfileName`). `internal/buildinfo/` — version/commit/date via ldflags.

## Conventions

- Auth: Bearer token + account domain; base URL `https://<domain>/rest/...`.
- Config: `~/.config/pfix/config.yml` (mode 0600), multiple named profiles. Precedence: flags > environment (`PFIX_DOMAIN`, `PFIX_TOKEN`, `PFIX_PROFILE`, `PFIX_CONFIG`) > config file. Resolve the profile name only through `config.ResolveProfileName` (`flag > PFIX_PROFILE > current_profile > "default"`).
- Output: lists render as a table, single objects as key/value detail. `--json` emits the API response unmodified (pretty-printed) — always through `output.EmitJSON`, the one place `--jq` is applied; never call `output.JSON` from a command. `--fields` overrides both the requested fields and the columns. `-q` drops the header row (lists) or prints only the affected id (writes). Errors go to stderr with a non-zero exit. Commands render generically from decoded `map[string]any` via dot-paths, not typed structs, so unconfirmed nested shapes need no model.
- `--fields` help: each read command lists its selectable fields from a co-located `xxxAvailableFields` constant (via `cmdutil.FieldsHelp`), taken from the API's OpenAPI description. It is published at `https://help.planfix.com/restapidocs/swagger.json` (public; the per-account `/restapidocs/swagger.json` redirects to a login page) and is not committed. When the spec's version changes, re-check the constants against it; where the spec leaves a list endpoint un-enumerated, reuse the sibling `view` vocabulary.
- Transport: `Client.Do`/`JSON` return the response for any status (use `planfix.ParseError` for detail) and retry connection errors and 5xx, never 4xx; `DoOnce`/`JSONOnce`/`Upload` make a single attempt. A retry is wrong whenever the first attempt may already have been applied: it duplicates a create, or reports a completed delete/restore as a failure. Never send an `Idempotency-Key` (a reused key is rejected with `409 code 43`, not replayed). Every request sends `User-Agent: pfix/<version>`; a caller-supplied header wins.
  - Single attempt: every create (`task`/`project`/`contact create`, `task comment add`, `task checklist add`), every delete, `contact restore`, `file upload` (local and `--from-url`).
  - Retrying: reads (including POST list endpoints) and idempotent writes — updates, `comment edit`, `task checklist update`, `file rename`, file attach (re-attaching is a no-op).
  - `api`: retries only `GET`/`HEAD`/`OPTIONS`; any other method is sent once, since a raw path does not say whether a POST creates.
- Proxy: every HTTP path resolves its proxy through the `Client.Proxy` field (default `http.ProxyFromEnvironment`) by building transports with `newTransport`. Never use a bare `http.Transport` — it silently disables proxying. `ALL_PROXY` is not honored (Go stdlib limitation).
- Updates send only the fields the user set (the API applies partial updates); an update with nothing to change refuses instead of posting an empty body.
- Destructive commands (`task delete`, `contact delete`, `file delete`, `comment delete`, `task checklist delete`): refuse to run without `--force` and never prompt, even interactively — behavior stays identical in a terminal and a script, with no TTY branch to keep in sync. Follow this for any future destructive command.
- Validate client-side whenever the API would accept bad input silently (see below), and say so in the flag help when a silent behavior cannot be detected.

## API behavior to design around

Each item below was verified live against the API. These are the traps that shape the code.

**Silent acceptance.** Many writes answer `success` while discarding or mangling input:
- Unknown `fields` names are ignored, so `--fields` help is advisory. Unknown keys in a checklist or file write *body*, by contrast, are rejected (`code 30`, *Cannot deserialize*).
- An invalid task `priority` is reset to `NotUrgent` — pfix validates it.
- A custom-field list value outside its `enumValues` is stored verbatim as a phantom option — pfix checks labels exactly (no trimming or case folding). A custom field the task's template does not carry is either refused with `code 6` (*Custom field not permitted by id*) or accepted and dropped silently (seen on a templated task) — verify by reading back. Binding a field to a template is not possible over REST (templates are read-only), so round-trips need fields bound in the web UI. Even on a correctly bound field, the API **intermittently refuses** writes with that same `code 6` (about 1 in 4–6 in a burst of updates); a refused write is not applied, so a retry is safe. `--cf <id>=` clears with the spec's per-type marker (`null`/`[]`/`false`) — verified live for text (reads back `""`), number and list; computed and unlisted type codes are refused.
- An unresolvable people reference (e.g. checklist `assignees`) is dropped; since lists are replaced wholesale, a bad reference clears the list.
- A task's `isDeleted` is read-only; writing it is dropped.
- `--from-url` stores whatever the URL returns, a 404 page included.

**Parent segments in routes.** Reads ignore them; some writes validate them; one write ignores them:
- Comment update ignores its parent segment entirely (any task or contact id works), so `comment edit` resolves the real parent with `GET /comment/{id}?fields=task,contact` first. `/comment/{id}` itself refuses `POST`/`PUT` (`405`, HTML body → parses as `code 0`).
- Checklist item `GET` ignores the task segment; update `POST` and `DELETE` validate it (`code 6` wrong task, `code 1000` unknown task). Commands take and forward both ids.

**Responses that need decoding beyond `result`.**
- Checklist update answers `200 success` with a `failures` array — decode it and fail on a non-empty one.
- Deletes can answer `200` with `result: "fail"` (seen on file delete) — run delete/restore responses through `cmdutil.CheckResult`. Send deletes and restores with `JSONOnce`: a retry after one the server carried out answers "not found" (`1000`/`2000`) or "not deleted" (`2002`).
- `report view` arrives under the misspelled key `repost` (fall back to `report`). Envelope keys are not always the resource name (`items`/`item` for checklists, `dataTags`/`dataTag`, `customFieldTypes`).
- File `size` is `ceil(bytes / 1024)`, not bytes — use `Content-Length` for exact sizes.

**Per-resource facts.**
- Tasks: delete is irreversible (no restore endpoint) and does not cascade to subtasks — they survive as top-level tasks with `parent` removed; checklist items are deleted; comments and files survive, unlinked. A task's process is fixed at creation; Draft (`--status 0`) is create-only; dates are `dd-MM-yyyy`/`HH:mm` in the account timezone; `GET /process/task` lists only custom processes (the built-in one is reachable through a task's `processId`). `--saved-filter` and `--filter` combine as AND.
- Contacts: `contact create` needs a template. Delete is restorable — `POST /contact/restore/` `{"id":N}` brings it back under the same id; restoring one that is not deleted fails with `2002`. `description` is plaintext on contacts but HTML on tasks/projects — the inline-file scrape reads a contact's comment feed instead.
- Users: create is disabled by the API; `user update` is declined. Only `user/groups` resolves `group:N` assignee references.
- Comments: a task's description seeds comment #1 and then diverges — edits never re-sync either way, and that first comment cannot be deleted (`code 0`). Deletion is soft (still listed under `--include-deleted`, no restore). `POST /comment/list` requires a `filters` array.
- Checklists: task-only; items nest via `parent` (validated); a checklist item is a task record that `GET /task/{id}` refuses. Delete is hard and cascades to nested items.
- Files: inline editor uploads appear in no `files[]` array — find them by scraping `uniqueid` from description/comment HTML. An object's `files` field is description-scoped. Uploads: 100 MiB hard cap (checked locally — the API rejects only after the full transfer), filename stored verbatim including path separators (send a base name, validate `--name`), same name + content re-uploaded within minutes returns the same id, attach target id is a query parameter. Delete (`DELETE /file/{id}`) detaches the file from every object but does not erase it — `GET /file/{id}` and the download still work, and re-attaching answers success without reattaching. A deleted file answers `200` `result: "fail"` `code 7000` ("has already been deleted") to a second delete or a rename; an unknown id is `400 code 7000` ("not found"). Rename (`POST /file/{id}` `{"name"}`) stores the name verbatim — path separators and an empty string included — so validate it as for uploads. The `link` key is accepted but has no visible effect on an uploaded file.
- Auth: `GET /ping` is the scope-independent token check. A rejected token is not always `401 code 1`: an account that no longer exists answered a bare `401` with no JSON envelope. The root (`cmd.Execute`) passes every command's error through `cmdutil.DescribeAPIError` (hints for code 1, code 5 and a code-less `401`, naming `--profile` or a `PFIX_TOKEN` override), so commands return API errors as they are; only command-specific hints live in the command. `GET /userinfo` (identity behind the token) has **no `result` envelope** — the profile fields are top-level, so never gate it on `result`; `groups` is absent for contact logins and `name`/`email`/`picture` may be null; the spec documents `403` for an unsupported login type or profile (not reproduced live). Keep it informational in `auth status`: its failure must not change that command's exit status.

## Toolchain

- Use the project's Go toolchain (latest stable); the `go` directive is pinned in `go.mod`. Keep the module graph tidy — `go mod tidy` must leave `go.mod`/`go.sum` unchanged, and every imported dependency belongs in the direct `require` block.
- Format: `gofmt -l .` (must print nothing). Vet: `go vet ./...`.
- Test: `go test ./...` (CI runs `-race`). Table-driven tests; fake the API with `net/http/httptest`; mock only at the HTTP boundary, never the code under test.
- Lint (required — CI gates on it): `golangci-lint run ./...` must report `0 issues` before work is done or a release is tagged. `gofmt`/`go vet` are not a substitute — `errcheck` flags unchecked error returns, including in tests. Config in `.golangci.yml`.
- Build: `go build -o pfix ./cmd/pfix`; release builds embed metadata via `-ldflags "-X github.com/a68366/pfix-cli/internal/buildinfo.Version=..."` (and `Commit`/`Date`).
- CI: `.github/workflows/ci.yml` runs gofmt/vet/build/`go test -race`/tidy-check/golangci-lint on pushes to `main` and PRs; `.github/workflows/release.yml` + `.goreleaser.yml` publish multi-platform binaries to GitHub Releases on `v*` tags.
- Releasing: on a green `main`, `git tag -a vX.Y.Z -m "pfix vX.Y.Z" && git push origin vX.Y.Z`. GoReleaser builds linux/darwin/windows × amd64/arm64, writes `checksums.txt`, and publishes the release immediately, with a changelog built from Conventional Commit subjects — `docs`/`test`/`chore`/`ci` are excluded, so a user-visible change needs `feat`/`fix`. Verify the Actions run and the Releases page afterwards.

## Testing rules

- All behaviour changes must be covered by tests — if it isn't tested, it isn't done.
- Test decisions and branches (error mapping, config precedence, retry behavior, request building), not glue code.

## Roadmap

- **Next (as API access allows):** a typed `directory` resource; running saved reports (`report` covers definitions only).
- **Postponed — not exposed via REST:** `process` mutation and running processes / workflow actions.
- **Declined by the user:** deletes (except `task delete`, `contact delete` (+ `contact restore`), `file delete`, `comment delete` and `task checklist delete`, all shipped at the user's request), `user update`, typed filter flags, and color.
