package comment

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

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
		Short: "Delete a comment",
		Long: "Delete a comment.\n\n" +
			"Deleting requires --force. The API deletes softly: the comment vanishes\n" +
			"from the default listing and can no longer be viewed or edited, but still\n" +
			"appears under 'pfix task comment list --include-deleted'. There is no way\n" +
			"to restore it. A task's first comment carries the task description and\n" +
			"cannot be deleted at all.",
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
		return fmt.Errorf("refusing to delete comment %d without --force", id)
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "DELETE", "comment/"+strconv.Itoa(id), nil)
	if err != nil {
		return describeDeleteError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", id)
		return nil
	}
	fmt.Fprintf(o.out, "Deleted comment %d\n", id)
	return nil
}

// describeDeleteError names the one known cause of the API's otherwise opaque
// "Rest API error" (app code 0) on this endpoint: a task's first comment holds
// the task description and is undeletable. Every other error, including the
// code 5000 not-found, falls through to the shared auth-hint mapping.
func describeDeleteError(err error) error {
	var apiErr *planfix.APIError
	if errors.As(err, &apiErr) && apiErr.Code == 0 && apiErr.StatusCode == http.StatusBadRequest {
		return fmt.Errorf("%w — a task's first comment holds its description and cannot be deleted", err)
	}
	return cmdutil.DescribeAPIError(err)
}
