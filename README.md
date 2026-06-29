# pfix

A command-line client for the [Planfix](https://planfix.com) REST API, written in Go. It ships as a single self-contained binary and is built for two audiences: people working in a terminal, and automation or AI agents that consume machine-readable output.

> **Status: early development.** Credential management (`auth`) and a raw, authenticated request passthrough (`api`) are implemented. Typed resource commands (`task`, `project`) and human-readable table output are on the [roadmap](#roadmap).

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

## Roadmap

- Typed `task` commands (list / view / create / update, plus comments), then `project`.
- Human-readable table output — the current `api` output is raw JSON; `--json` selects raw output once typed commands ship.
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
