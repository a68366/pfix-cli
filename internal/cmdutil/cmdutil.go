package cmdutil

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/a68366/pfix-cli/internal/config"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// GlobalOpts holds persistent flags shared by every subcommand.
type GlobalOpts struct {
	Profile string
	Domain  string
	JSON    bool
	Fields  string
	Quiet   bool
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
		if err != nil || id <= 0 {
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
