package config

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	pfconfig "github.com/a68366/pfix-cli/internal/config"
)

func newUseCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "use <name>",
		Short: "Set the active profile",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath()
			if err != nil {
				return err
			}
			return runUse(path, args[0], cmd.OutOrStdout(), g.Quiet)
		},
	}
}

func runUse(path, name string, out io.Writer, quiet bool) error {
	cfg, err := pfconfig.Load(path)
	if err != nil {
		return err
	}
	if _, ok := cfg.Profiles[name]; !ok {
		return fmt.Errorf("no such profile: %s", name)
	}
	cfg.CurrentProfile = name
	if err := pfconfig.Save(path, cfg); err != nil {
		return err
	}
	if !quiet {
		fmt.Fprintf(out, "Switched to profile %q\n", name)
	}
	return nil
}
