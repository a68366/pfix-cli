package cmdutil

import (
	"os"

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
