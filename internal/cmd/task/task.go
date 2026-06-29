package task

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `task` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Work with Planfix tasks",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g), newCreateCmd(g), newUpdateCmd(g), newCommentCmd(g))
	return cmd
}
