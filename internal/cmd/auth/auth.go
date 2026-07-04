package auth

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `auth` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "auth",
		Short: "Authenticate pfix with a Planfix account",
	}
	cmd.AddCommand(newLoginCmd(g), newStatusCmd(g), newLogoutCmd(g))
	return cmd
}
