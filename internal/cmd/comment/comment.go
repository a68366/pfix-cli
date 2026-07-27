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

// AvailableFields is the comment field vocabulary, shared by every command
// that reads comments: this group's `view` and the task-scoped
// `pfix task comment list`. Both read the same object, so both offer the same
// --fields names.
const AvailableFields = "id,dateTime,type,fromType,description,additionalDescriptionData,task,project,contact,owner,isDeleted,isPinned,isHidden,isNotRead,recipients,reminders,dataTags,files,changeTaskStartDate,changeTaskExpectDate,changeStatus,sourceObjectId,sourceDataVersion"

// NewCmd builds the `comment` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Work with Planfix comments",
		Long: "Work with Planfix comments.\n\n" +
			"A comment is a global object, not a child of the task or contact it hangs\n" +
			"off: view/edit/delete here take the comment's own id (as printed by\n" +
			"'pfix task comment list'). Listing and adding are done through the parent\n" +
			"object instead — see 'pfix task comment list <task-id>' and\n" +
			"'pfix task comment add <task-id>'.",
	}
	cmd.AddCommand(newViewCmd(g), newEditCmd(g), newDeleteCmd(g))
	return cmd
}
