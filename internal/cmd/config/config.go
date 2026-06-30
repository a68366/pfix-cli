package config

import (
	"os"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	pfconfig "github.com/a68366/pfix-cli/internal/config"
)

// NewCmd returns the "config" command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage pfix configuration profiles",
	}
	cmd.AddCommand(newListCmd(g))
	cmd.AddCommand(newUseCmd(g))
	cmd.AddCommand(newShowCmd(g))
	return cmd
}

// configPath resolves the active config file path.
func configPath() (string, error) {
	return pfconfig.DefaultPath(os.Getenv)
}
