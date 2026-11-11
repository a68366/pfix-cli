package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type deleteOptions struct {
	force  bool
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newDeleteCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &deleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a file, removing it from everything it is attached to",
		Long: "Delete a file.\n\n" +
			"Deleting requires --force, and it cannot be undone.\n\n" +
			"The file is removed from every task, contact, and project it is attached\n" +
			"to, but the API does not erase it: it stays viewable and downloadable by\n" +
			"id with 'pfix file view' and 'pfix file download'. A deleted file cannot be\n" +
			"renamed, and it cannot be attached again — the API accepts the attach\n" +
			"request, but the file does not reappear on the object.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runDelete(cmd.Context(), o, args[0])
		},
	}
	cmd.Flags().BoolVar(&o.force, "force", false, "Confirm the deletion (required)")
	return cmd
}

func runDelete(ctx context.Context, o *deleteOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	if !o.force {
		return fmt.Errorf("refusing to delete file %d without --force", id)
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	// One attempt only: a retry after a delete the server already carried
	// out would answer "already deleted" and report success as a failure.
	raw, err := client.JSONOnce(ctx, http.MethodDelete, "file/"+strconv.Itoa(id), nil)
	if err == nil {
		// An already-deleted file answers HTTP 200 with result "fail".
		err = cmdutil.CheckResult(raw)
	}
	if err != nil {
		return describeFileWriteError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", id)
		return nil
	}
	fmt.Fprintf(o.out, "Deleted file %d\n", id)
	return nil
}

// describeFileWriteError explains code 7000, which delete and rename (the
// file update endpoint) answer both for an unknown id and for a file that is already
// deleted; the API's message tells the two apart. Everything else is returned
// as is (auth failures get their hint at the root).
func describeFileWriteError(err error) error {
	var apiErr *planfix.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != 7000 {
		return err
	}
	if strings.Contains(apiErr.Message, "already been deleted") {
		return fmt.Errorf("%w — a deleted file can still be viewed and downloaded, but not changed", err)
	}
	return fmt.Errorf("%w — no file with that id", err)
}
