// Package file implements the top-level `file` command group: view metadata for
// a single file (GET /file/{id}), download its bytes (GET /file/{id}/download),
// and upload one (POST /file/ or /file/from-url/), optionally attaching it to a
// task, contact, or project; rename one (POST /file/{id}); and delete one
// (DELETE /file/{id}), which detaches it everywhere but does not erase it.
package file

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewCmd builds the `file` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "file",
		Short: "Work with Planfix files",
	}
	cmd.AddCommand(newViewCmd(g), newDownloadCmd(g), newUploadCmd(g), newRenameCmd(g), newDeleteCmd(g))
	return cmd
}
