package contact

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/groups"
	"github.com/a68366/pfix-cli/internal/cmd/processes"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `contact` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "contact",
		Short: "Work with Planfix contacts",
	}
	cg := groups.NewCmd(g, "contact")
	cg.Short = "List contact groups (categories)"
	cg.Long = "List contact groups — the contact categories such as Клиент, Партнёр, Поставщик."
	cmd.AddCommand(newListCmd(g), newViewCmd(g), newCreateCmd(g), newUpdateCmd(g), processes.NewCmd(g, "contact"), cg)
	return cmd
}
