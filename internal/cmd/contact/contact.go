package contact

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/files"
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
	cg.Short = "List contact groups"
	cg.Long = groups.Long("List contact groups — the segments a contact base is divided into,\n" +
		"such as customers, partners or suppliers. These are not the group:N\n" +
		"references accepted by --assignees/--auditors/--participants; those are\n" +
		"user groups (see 'pfix user groups').")
	cmd.AddCommand(newListCmd(g), newViewCmd(g), newCreateCmd(g), newUpdateCmd(g), processes.NewCmd(g, "contact"), cg, files.NewCmd(g, files.Options{Type: "contact", DescriptionOnly: true}))
	return cmd
}
