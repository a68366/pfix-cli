// Package comment implements the top-level `comment` command group: view, edit,
// and delete one comment by its global id.
//
// A Planfix comment is a single entity with an optional parent link (a task or
// a contact), not a child of either. Only creating and listing are genuinely
// parent-scoped, and those stay on the parent's own group (`pfix task comment
// list|add`); everything here works on any comment regardless of what it hangs
// off.
package comment

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `comment` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Work with Planfix comments",
	}
	cmd.AddCommand(newViewCmd(g))
	return cmd
}
