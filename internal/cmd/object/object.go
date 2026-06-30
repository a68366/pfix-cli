package object

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `object` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "object",
		Short: "Work with Planfix objects",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g))
	return cmd
}
