package datatag

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `datatag` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "datatag",
		Short: "Work with Planfix data tags",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g))
	return cmd
}
