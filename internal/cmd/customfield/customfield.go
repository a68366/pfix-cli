package customfield

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `customfield` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "customfield",
		Short: "List Planfix custom-field definitions for an object type",
	}
	cmd.AddCommand(newListCmd(g))
	return cmd
}
