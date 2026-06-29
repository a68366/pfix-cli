package auth

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/config"
)

func newLogoutCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Remove stored credentials for a profile",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := config.DefaultPath(os.Getenv)
			if err != nil {
				return err
			}
			cfg, err := config.Load(path)
			if err != nil {
				return err
			}
			name := config.ResolveProfileName(g.Profile, os.Getenv, cfg)
			return runLogout(path, name, cmd.OutOrStdout())
		},
	}
}

func runLogout(path, name string, out io.Writer) error {
	cfg, err := config.Load(path)
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("no such profile: %s", name)
	}
	delete(cfg.Profiles, name)
	if cfg.CurrentProfile == name {
		cfg.CurrentProfile = ""
	}
	if err := config.Save(path, cfg); err != nil {
		return err
	}
	fmt.Fprintf(out, "Removed profile %q\n", name)
	return nil
}
