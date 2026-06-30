# pfix

A command-line client for the [Planfix](https://planfix.com) REST API, written in Go. It ships as a single self-contained binary and is built for two audiences: people working in a terminal, and automation or AI agents that consume machine-readable output.

> **Status: early development.** Credential management (`auth`), a raw authenticated request passthrough (`api`), and the typed `task`, `project`, `contact`, `user`, and `report` commands (with human-readable table output) are implemented. Further typed resources are on the [roadmap](#roadmap).

## Install

Requires Go 1.26 or newer.

Build from source:

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

For CI and automation, skip the config file and pass credentials via the environment:

```sh
export PFIX_DOMAIN=example.planfix.com
export PFIX_TOKEN=your-token
pfix api task/1
```

## Usage

Typed commands print a human-readable table (or a key/value detail block for a
single item) by default. Three global flags shape the output of every typed
command:

| Flag | Effect |
|---|---|
| `--json` | Emit the raw Planfix API response (pretty-printed) instead of a table — the machine-readable path |
| `--fields a,b,c` | Override which Planfix fields are requested and shown as columns (defaults are per-command) |
| `-q, --quiet` | Drop the table header (lists), or print only the affected id (`create`/`update`/`comment add`) |

### Tasks

```sh
# List tasks (table). --limit / --offset page the results.
pfix task list
pfix task list --limit 20 --offset 20
pfix task list --fields id,name,status      # choose your own columns
pfix task list --json                       # raw API response

# View one task (detail block, or --json for everything)
pfix task view 17
pfix task view 17 --json

# Create a task (--name required); prints the new id
pfix task create --name "Deploy release" --description "ship it"
pfix task create --name "Quick task" -q     # prints just the id

# Update a task — pass any of --name/--description/--status (status id)
pfix task update 17 --status 2
pfix task update 17 --name "Renamed" --description "new body"

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

- Further typed resources beyond `task` and `project`.
- Richer list filtering (currently use `api` with a POST body for arbitrary filters).
- Multi-platform release binaries.

## Development

```sh
go test ./...        # run the test suite
go vet ./...         # static analysis
gofmt -l .           # formatting check (prints nothing when clean)
go build -o pfix .   # build the binary
```

See [`CLAUDE.md`](CLAUDE.md) for architecture and conventions.

## License

[Apache-2.0](LICENSE).
