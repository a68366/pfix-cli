package task

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/files"
	"github.com/a68366/pfix-cli/internal/cmd/processes"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `task` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Work with Planfix tasks",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g), newCreateCmd(g), newUpdateCmd(g), newCommentCmd(g), newFiltersCmd(g), newStatusesCmd(g), processes.NewCmd(g, "task"), files.NewCmd(g, files.Options{Type: "task", DescriptionOnly: true}))
	return cmd
}
