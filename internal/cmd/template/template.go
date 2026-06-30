package template

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `template` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "template",
		Short: "List Planfix templates for an object type",
	}
	cmd.AddCommand(newListCmd(g))
	return cmd
}
