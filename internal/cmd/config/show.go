package config

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	pfconfig "github.com/a68366/pfix-cli/internal/config"
)

func newShowCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "show [name]",
		Short: "Show a profile's domain and masked token",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := configPath()
			if err != nil {
				return err
			}
			cfg, err := pfconfig.Load(path)
			if err != nil {
				return err
			}
			name := pfconfig.ResolveProfileName(g.Profile, os.Getenv, cfg)
			if len(args) == 1 {
				name = args[0]
			}
			return runShow(path, name, cmd.OutOrStdout())
		},
	}
}

func runShow(path, name string, out io.Writer) error {
	cfg, err := pfconfig.Load(path)
	if err != nil {
		return err
	}
	p, ok := cfg.Profiles[name]
	if !ok {
		return fmt.Errorf("no such profile: %s", name)
	}
	fmt.Fprintf(out, "Profile: %s\n", name)
	fmt.Fprintf(out, "Domain:  %s\n", p.Domain)
	fmt.Fprintf(out, "Token:   %s\n", cmdutil.MaskToken(p.Token))
	return nil
}
