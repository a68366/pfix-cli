# Contributing

Issues and pull requests are welcome. For anything non-trivial, please open an
issue first so the approach can be discussed before you invest time in code.

## Before you open a PR

Run the same checks CI runs:

```sh
gofmt -l .           # must print nothing
go vet ./...
go test ./...
go mod tidy          # must leave go.mod / go.sum unchanged
golangci-lint run    # optional locally; CI runs it
```

Ground rules:

- **Behaviour changes need tests.** Table-driven tests, with the API faked via
  `net/http/httptest`. Mock only at the HTTP boundary.
- **Keep dependencies lean and public.** Every dependency must come from a
  public source and be listed in the direct `require` block.
- **Commit messages** follow the Conventional Commits style used in the
  history: `feat(task): ...`, `fix: ...`, `docs: ...`, `test: ...`, `ci: ...`.

See [`AGENTS.md`](AGENTS.md) for the project layout, conventions, and roadmap.
