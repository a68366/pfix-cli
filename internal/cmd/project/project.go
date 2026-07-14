package project

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/files"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `project` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Work with Planfix projects",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g), newCreateCmd(g), newUpdateCmd(g), files.NewCmd(g, files.Options{Type: "project", Paging: true}))
	return cmd
}
