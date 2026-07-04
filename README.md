# pfix

[![CI](https://github.com/a68366/pfix-cli/actions/workflows/ci.yml/badge.svg)](https://github.com/a68366/pfix-cli/actions/workflows/ci.yml)

An unofficial command-line client for the [Planfix](https://planfix.com) REST API, written in Go. Ships as a single self-contained binary.

> **Unofficial.** pfix is an independent open-source project. It is **not** an official Planfix product and is not affiliated with, endorsed, sponsored, or funded by Planfix. The Planfix name is used only to describe the API this tool connects to.

> **Status:** functional and actively developed. Typed commands cover tasks, projects, contacts, users, reports, data tags, templates, custom fields, and objects; anything not covered yet is reachable through the raw `api` passthrough (remaining work is on the [roadmap](#roadmap)). Command and flag conventions may still change before v1.0.

## Install

**Prebuilt binaries:** download the archive for your platform from the [Releases page](https://github.com/a68366/pfix-cli/releases), unpack it, and put `pfix` on your `PATH`.

**Build from source** (requires Go 1.26 or newer):

```sh
git clone https://github.com/a68366/pfix-cli
cd pfix-cli
go build -o pfix .
```

Or via `go install` (installs to `$(go env GOPATH)/bin`):

```sh
go install github.com/a68366/pfix-cli@latest
```

> `go install` names the binary `pfix-cli`; rename it to `pfix` if you prefer the shorter command.

## Authenticate

`pfix` talks to a Planfix account using a REST API **token** (create one in your Planfix account settings) and your account **domain** (e.g. `example.planfix.com`).

Interactive login stores credentials in a config profile:

```sh
pfix auth login
# Planfix domain (e.g. example.planfix.com): example.planfix.com
# API token: ********

pfix auth status     # show the active profile and check the token
pfix auth logout     # remove a profile's credentials
```

Check connectivity and that the active token is accepted with `pfix ping` — it prints
`OK` on success (`--json` for the raw response, `-q` to print nothing and just set the
exit code) and exits non-zero if the token is rejected. It is the lightest such check;
`pfix auth status` uses the same `GET /ping` probe to validate the token.

For CI and automation, skip the config file and pass credentials via the environment:

```sh
export PFIX_DOMAIN=example.planfix.com
export PFIX_TOKEN=your-token
pfix api task/1
```

## Usage

Typed commands print a human-readable table (or a key/value detail block for a
single item) by default. A handful of global flags shape the output of every
typed command:

| Flag | Effect |
|---|---|
| `--json` | Emit the raw Planfix API response (pretty-printed) instead of a table — the machine-readable path |
| `--jq '<expr>'` | Filter the JSON output through a jq expression, one result per line (implies `--json`) |
| `--fields a,b,c` | Override which Planfix fields are requested and shown as columns (defaults are per-command) |
| `-q, --quiet` | Drop the table header (lists), or print only the affected id (`create`/`update`/`comment add`) |

**Reshaping JSON with `--jq`.** `--jq` runs a jq expression over the same JSON
`--json` would print, so you don't need to pass both. A result that is a bare
string prints raw and unquoted (pipe-friendly); any other result (object,
array, number, bool, null) prints as compact JSON, one result per line. An
invalid expression is rejected before any request is made. The jq engine is
embedded in `pfix` — no external `jq` binary is required.

**Filtering lists.** The `list` commands for `task`, `project`, `contact`, `user`,
`report`, `datatag`, and `object` accept `--filter <json>` — a raw Planfix filters
array forwarded to the API:

```sh
pfix task list --filter '[{"type":51,"operator":"equal","value":42}]'
```

Filter `type` codes are Planfix-specific; see the Planfix REST filter reference for
the available types and operators.

**Saved filters (task only).** Instead of a raw array, `task list` can apply one of
the account's saved filters by id with `--saved-filter`. List the available filters —
system ones (`:all`, `:in`, `:out`, `:audit`) and user-defined views — with `task filters`:

```sh
pfix task filters                 # table of saved task filters (ID / NAME / OWNER)
pfix task list --saved-filter :in # apply the "Incoming" saved filter
pfix task list --saved-filter 220612
```

A saved-filter id is an opaque string. `--filter` and `--saved-filter` combine as a
logical AND — the raw filter further narrows the saved view (both constraints apply
together, verified against the API).

### Tasks

```sh
# List tasks (table). --limit / --offset page the results.
pfix task list
pfix task list --limit 20 --offset 20
pfix task list --fields id,name,status      # choose your own columns
pfix task list --json                       # raw API response
pfix task list --saved-filter :in           # apply a saved filter (see: pfix task filters)
pfix task filters                           # list saved task filters
pfix task list --jq '.tasks[].name'         # just the names, one per line

# View one task (detail block, or --json for everything)
pfix task view 17
pfix task view 17 --json
pfix task view 17 --jq '.task.status.name'  # a single value, unquoted

# Create a task (--name required); prints the new id
pfix task create --name "Deploy release" --description "ship it"
pfix task create --name "Release prep" --template 6 --project 21 \
  --assignees user:1,contact:4 --auditors group:3 --end-date 2026-07-20
pfix task create --name "Quick task" -q     # prints just the id

# Update a task — pass any field flag (people lists and dates are replaced, not merged)
pfix task update 17 --status 2
pfix task update 17 --name "Renamed" --description "new body"
pfix task update 17 --assignees user:1 --priority Urgent --start-date "2026-07-08 10:00"

# Comments
pfix task comment list 17
pfix task comment add 17 --body "Looks good"
echo "comment from stdin" | pfix task comment add 17
```

Notes:
- A task's **description** is its first comment in Planfix — it shows up in
  `comment list` as well as in `view`.
- `--status` takes a numeric status id (see a task's current status via
  `pfix task view <id> --json`). Field names for `--fields` are Planfix REST
  field names; unknown names are silently ignored by the API.
- `--assignees`/`--auditors`/`--participants` take comma-separated prefixed
  references — `user:N`, `contact:N`, or `group:N`. On `update` the list you
  pass **replaces** the stored one.
- `--start-date`/`--end-date` accept `YYYY-MM-DD`, `"YYYY-MM-DD HH:MM"`, or
  `YYYY-MM-DDTHH:MM`; Planfix interprets the time in the account's timezone.
- `--priority` is `Urgent` or `NotUrgent` (validated locally — the API would
  silently fall back to `NotUrgent` on anything else).
- `--counterparty` takes a contact id or `contact:N`. `--template` exists only
  on `create`; a task's template cannot be changed afterwards.

### Projects

The same shape as `task`, for Planfix projects (projects have no comments):

```sh
pfix project list                              # table; --limit / --offset page
pfix project list --fields id,name,owner       # choose columns
pfix project view 12                           # detail block (--json for everything)
pfix project create --name "Q3 Launch"         # prints the new id
pfix project update 12 --name "Q3 Launch v2" --status 2
```

### Contacts

Planfix contacts (people and companies). Note: `contact create` requires a
template id — find yours with `pfix api contact/list --fields template` or in the
Planfix UI.

```sh
pfix contact list                                   # table; --limit / --offset page
pfix contact view 42                                # detail block (--json for everything)
pfix contact create --name "Ada" --lastname "Lovelace" --template 1 --email ada@example.com
pfix contact update 42 --email new@example.com --lastname "Byron"
```

### Users

Planfix staff/users, read-only (the API does not allow creating users):

```sh
pfix user list                       # table; --limit / --offset page
pfix user view 1                     # detail block (--json for everything)
pfix user list --fields id,name,role # choose columns
```

### Reports

Planfix saved reports, read-only (definitions; running a report is not yet supported):

```sh
pfix report list            # table of saved reports
pfix report view 209428     # report definition (--json for the full column list)
```

### Templates

List the templates available for an object type (read-only):

```sh
pfix template list task         # task templates
pfix template list contact      # contact templates (people + companies)
pfix template list project
```

### Objects

Planfix objects, read-only:

```sh
pfix object list            # table; --limit / --offset page
pfix object view 1          # detail block (--json for everything)
```

### Custom fields

List the custom-field definitions for an object type (read-only):

```sh
pfix customfield list task        # custom fields defined on tasks
pfix customfield list contact     # (empty if none are defined)
```

### Data tags

Planfix data tags (custom structured-data record types), read-only:

```sh
pfix datatag list           # table of data tags
pfix datatag view 4         # a tag's definition (--json for its field list)
```

### Raw API passthrough

`pfix api <path>` makes an authenticated request to any Planfix REST endpoint and prints the raw JSON response — handy for endpoints without a dedicated command yet, and for scripting.

```sh
# GET a task
pfix api task/1

# POST a JSON body from stdin — the primary path for Planfix's nested request bodies
echo '{"pageSize":5,"fields":"id,name"}' | pfix api task/list --input -

# Set simple typed parameters (auto-switches the method to POST)
pfix api task/ -F name="Deploy" -F template=42

# Add a header, include the response status/headers in the output
pfix api task/1 -H "X-Custom: value" -i
```

Flags:

| Flag | Meaning |
|---|---|
| `-X, --method` | HTTP method (default `GET`, or `POST` when a body/fields are supplied) |
| `-F, --field key=value` | Typed parameter: integers, `true`/`false`/`null`, or `@file`/`@-` (file/stdin) |
| `-f, --raw-field key=value` | String parameter |
| `--input <file\|->` | Send a raw request body (`-` reads stdin) |
| `-H, --header key:value` | Add a request header |
| `-i, --include` | Print the response status and headers |
| `--silent` | Suppress the response body |

A non-2xx response prints the body, then exits non-zero with the API error message on stderr.

## Configuration

Credentials live in `~/.config/pfix/config.yml` (mode `0600`) and support multiple named **profiles**:

```yaml
current_profile: default
profiles:
  default:
    domain: example.planfix.com
    token: "..."
  staging:
    domain: staging.planfix.com
    token: "..."
```

Choose a profile per command with `--profile staging` or `PFIX_PROFILE=staging`. Resolution precedence is **flags > environment > config file**; the environment variables are `PFIX_DOMAIN`, `PFIX_TOKEN`, `PFIX_PROFILE`, and `PFIX_CONFIG` (overrides the config file path).

Manage profiles without editing the file by hand:

```sh
pfix config list            # table of profiles; * marks the active one
pfix config show            # active profile's domain + masked token (or: config show <name>)
pfix config use staging     # set the active profile (current_profile)
```

## Roadmap

- Typed `directory` and `file` commands.
- Running saved reports (`report` currently covers definitions only).

## Development

```sh
go test ./...        # run the test suite
go vet ./...         # static analysis
gofmt -l .           # formatting check (prints nothing when clean)
go build -o pfix .   # build the binary
golangci-lint run    # lint (optional locally; CI runs it)
```

See [`AGENTS.md`](AGENTS.md) for architecture and conventions.

## License

[Apache-2.0](LICENSE).
