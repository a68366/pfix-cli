package cmdutil

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

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
	return id, nil
}

// DecodeJSON unmarshals b into v, wrapping any error with context.
func DecodeJSON(b []byte, v any) error {
	if err := json.Unmarshal(b, v); err != nil {
		return fmt.Errorf("decode response: %w", err)
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
