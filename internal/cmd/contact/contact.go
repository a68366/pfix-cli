package contact

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/processes"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `contact` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contact",
		Short: "Work with Planfix contacts",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g), newCreateCmd(g), newUpdateCmd(g), processes.NewCmd(g, "contact"))
	return cmd
}
