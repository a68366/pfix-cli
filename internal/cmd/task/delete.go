package task

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
		Short: "Delete a task",
		Long: "Delete a task.\n\n" +
			"Deleting requires --force, and it cannot be undone: the API has no way\n" +
			"to restore a deleted task.\n\n" +
			"What goes with it, and what stays:\n" +
			"  - its checklist items are deleted with it;\n" +
			"  - its subtasks are NOT deleted — they survive as top-level tasks, no\n" +
			"    longer linked to any parent (delete them first if they should go too);\n" +
			"  - its comments and attached files stay in place, no longer linked to\n" +
			"    the task.",
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
		return fmt.Errorf("refusing to delete task %d without --force", id)
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	// One attempt only: if a retried DELETE followed one the server had
	// already carried out, it would answer "not found" and report a
	// successful delete as a failure.
	raw, err := client.JSONOnce(ctx, http.MethodDelete, "task/"+strconv.Itoa(id), nil)
	if err == nil {
		err = cmdutil.CheckResult(raw)
	}
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
	fmt.Fprintf(o.out, "Deleted task %d\n", id)
	return nil
}

// describeDeleteError explains code 1000: the API answers it both for an id
// that never existed and for a task that is already deleted. Everything else
// is returned as is (auth failures get their hint at the root).
func describeDeleteError(err error) error {
	var apiErr *planfix.APIError
	if errors.As(err, &apiErr) && apiErr.Code == 1000 {
		return fmt.Errorf("%w — there is no task with that id, or it was already deleted", err)
	}
	return err
}
