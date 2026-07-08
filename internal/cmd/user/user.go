package user

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/groups"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `user` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "user",
		Short: "Work with Planfix users",
	}
	ug := groups.NewCmd(g, "user")
	ug.Short = "List user (employee) groups"
	ug.Long = "List user (employee) groups — the group:N values accepted by --assignees/--auditors/--participants."
	cmd.AddCommand(newListCmd(g), newViewCmd(g), ug, newPositionsCmd(g))
	return cmd
}
