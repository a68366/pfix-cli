# Security Policy

## Reporting a vulnerability

Please do not report security issues in public GitHub issues. Instead, use
GitHub's private vulnerability reporting on this repository (**Security** tab →
**Report a vulnerability**). Reports are handled on a best-effort basis.

## Handling credentials

- `pfix` stores API tokens in `~/.config/pfix/config.yml` with file mode `0600`.
  Treat that file like a password.
- Tokens can also be supplied via the `PFIX_TOKEN` environment variable; be
  careful not to leak it into shell history, CI logs, or process listings.
- `pfix auth status` and `pfix config show` print tokens masked, never in full.
- A Planfix API token grants whatever access it was created with — prefer
  narrowly-scoped tokens where your Planfix plan allows it, and revoke tokens
  you no longer use.
