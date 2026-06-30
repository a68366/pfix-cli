package report

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `report` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "report",
		Short: "Work with Planfix reports",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g))
	return cmd
}
