// Package file implements the top-level `file` command group: view metadata for
// a single file (GET /file/{id}), download its bytes (GET /file/{id}/download),
// and upload one (POST /file/ or /file/from-url/), optionally attaching it to a
// task, contact, or project.
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
	cmd.AddCommand(newViewCmd(g), newDownloadCmd(g), newUploadCmd(g))
	return cmd
}
