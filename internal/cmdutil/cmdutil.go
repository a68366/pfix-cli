package cmdutil

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/a68366/pfix-cli/internal/config"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// GlobalOpts holds persistent flags shared by every subcommand.
type GlobalOpts struct {
	Profile string
	Domain  string
	JSON    bool
	Fields  string
	Quiet   bool
	JQ      string
}

// PreRun applies global-flag interactions before a command runs. A non-empty
// --jq turns on JSON output and is validated up front, so an invalid
// expression fails before any API call is made.
func (g *GlobalOpts) PreRun() error {
	if g.JQ == "" {
		return nil
	}
	g.JSON = true
	if _, err := output.CompileJQ(g.JQ); err != nil {
		return err
	}
	return nil
}

// FieldsCSV returns override if non-empty, otherwise returns def.
func FieldsCSV(override, def string) string {
	if override != "" {
		return override
	}
	return def
}

// ValidateID parses a string ID and returns an informative error on failure.
func ValidateID(idStr string) (int, error) {
	id, err := strconv.Atoi(idStr)
	if err != nil {
		return 0, fmt.Errorf("id must be a number, got %q", idStr)
	}
	if id <= 0 {
		return 0, fmt.Errorf("id must be a positive number, got %d", id)
	}
	return id, nil
}

// DecodeJSON unmarshals b into v, wrapping any error with context.
func DecodeJSON(b []byte, v any) error {
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

// ValidateObjectType checks an object-type path segment (e.g. "task"): non-empty,
// lowercase ASCII letters only. Returns a clear error otherwise.
func ValidateObjectType(t string) error {
	if t == "" {
		return fmt.Errorf("object type is required (e.g. task, project, contact)")
	}
	for _, r := range t {
		if r < 'a' || r > 'z' {
			return fmt.Errorf("invalid object type %q: use lowercase letters (e.g. task, project, contact)", t)
		}
	}
	return nil
}

// MaskToken renders a token as **** plus its last 4 chars (or just **** when short).
func MaskToken(t string) string {
	if len(t) <= 4 {
		return "****"
	}
	return "****" + t[len(t)-4:]
}

// ClientFunc returns a zero-argument func that builds and returns a configured
// Planfix client. It is used to defer client construction until a command runs.
func (g *GlobalOpts) ClientFunc() func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c, _, err := g.Client()
		return c, err
	}
}

// ApplyFilter parses the --filter JSON (a Planfix filters value, usually an array)
// and, when non-empty, sets body["filters"]. Empty filter is a no-op. Invalid JSON
// returns a clear error.
func ApplyFilter(body map[string]any, filter string) error {
	if filter == "" {
		return nil
	}
	var v any
	if err := json.Unmarshal([]byte(filter), &v); err != nil {
		return fmt.Errorf("invalid --filter JSON: %w", err)
	}
	body["filters"] = v
	return nil
}

// Client builds a Planfix client from config, applying flag and env overrides.
func (g *GlobalOpts) Client() (*planfix.Client, config.Resolved, error) {
	path, err := config.DefaultPath(os.Getenv)
	if err != nil {
		return nil, config.Resolved{}, err
	}
	cfg, err := config.Load(path)
	if err != nil {
		return nil, config.Resolved{}, err
	}
	res, err := config.Resolve(cfg, config.Overrides{Profile: g.Profile, Domain: g.Domain}, os.Getenv)
	if err != nil {
		return nil, res, err
	}
	return planfix.New(res.Domain, res.Token), res, nil
}

// ParsePeople converts prefixed people references (user:N, contact:N, group:N)
// into the Planfix people-list shape:
// {"users": [{"id": "user:N"}, ...], "groups": [{"id": N}, ...]}.
// user: and contact: refs keep their string form in "users"; group: refs
// become int ids in "groups". Order is preserved; values are not deduplicated.
func ParsePeople(refs []string) (map[string]any, error) {
	users := []any{}
	groups := []any{}
	for _, ref := range refs {
		kind, num, found := strings.Cut(ref, ":")
		if !found || (kind != "user" && kind != "contact" && kind != "group") {
			return nil, fmt.Errorf("invalid people reference %q: use user:N, contact:N, or group:N", ref)
		}
		id, err := strconv.Atoi(num)
		if err != nil || id <= 0 || num != strconv.Itoa(id) {
			return nil, fmt.Errorf("invalid people reference %q: id must be a positive number", ref)
		}
		if kind == "group" {
			groups = append(groups, map[string]any{"id": id})
		} else {
			users = append(users, map[string]any{"id": ref})
		}
	}
	return map[string]any{"users": users, "groups": groups}, nil
}

// ParseTimePoint parses an ISO date or datetime into the Planfix time-point
// shape {"date": "dd-MM-yyyy"} or {"date": ..., "time": "HH:mm"}. Accepted
// inputs: 2006-01-02, "2006-01-02 15:04", 2006-01-02T15:04. Planfix
// interprets the wall-clock value in the account's timezone.
func ParseTimePoint(s string) (map[string]any, error) {
	if t, err := time.Parse("2006-01-02", s); err == nil {
		return map[string]any{"date": t.Format("02-01-2006")}, nil
	}
	for _, layout := range []string{"2006-01-02 15:04", "2006-01-02T15:04"} {
		if t, err := time.Parse(layout, s); err == nil {
			return map[string]any{"date": t.Format("02-01-2006"), "time": t.Format("15:04")}, nil
		}
	}
	return nil, fmt.Errorf(`invalid date %q: use YYYY-MM-DD, "YYYY-MM-DD HH:MM", or YYYY-MM-DDTHH:MM`, s)
}

// DescribeAPIError augments a Planfix *APIError with an actionable hint for the
// two common auth failures: an unknown token (app code 1) versus a valid token
// that lacks the scope for the requested action (app code 5). Any other error —
// including a nil error or a non-APIError — is returned unchanged.
func DescribeAPIError(err error) error {
	var apiErr *planfix.APIError
	if !errors.As(err, &apiErr) {
		return err
	}
	switch apiErr.Code {
	case 1:
		return fmt.Errorf("%w — the token was rejected; run `pfix auth login` to re-authenticate", err)
	case 5:
		return fmt.Errorf("%w — the token is valid but lacks the scope for this action", err)
	}
	return err
}
