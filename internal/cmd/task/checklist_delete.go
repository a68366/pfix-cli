package task

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type checklistDeleteOptions struct {
	force  bool
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newChecklistDeleteCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &checklistDeleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete <task-id> <item-id>",
		Short: "Delete a checklist item",
		Long: "Delete a checklist item.\n\n" +
			"Deleting requires --force. Deleting an item also deletes every item\n" +
			"nested under it, and it cannot be undone: the API has no way to\n" +
			"restore a deleted item.\n\n" +
			"Like update, a delete checks the pair: the item must belong to the\n" +
			"task id in the route, or the API answers 400.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runChecklistDelete(cmd.Context(), o, args[0], args[1])
		},
	}
	cmd.Flags().BoolVar(&o.force, "force", false, "Confirm the deletion (required)")
	return cmd
}

func runChecklistDelete(ctx context.Context, o *checklistDeleteOptions, taskArg, itemArg string) error {
	taskID, err := cmdutil.ValidateID(taskArg)
	if err != nil {
		return err
	}
	itemID, err := cmdutil.ValidateID(itemArg)
	if err != nil {
		return err
	}
	if !o.force {
		return fmt.Errorf("refusing to delete checklist item %d without --force", itemID)
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, http.MethodDelete, checklistItemPath(taskID, itemID), nil)
	if err != nil {
		return describeChecklistDeleteError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", itemID)
		return nil
	}
	fmt.Fprintf(o.out, "Deleted checklist item %d from task %d\n", itemID, taskID)
	return nil
}

// describeChecklistDeleteError hints at the cause behind the two app codes
// whose message alone can mislead. Code 600 (item not found) is also what a
// repeated delete answers, and what a nested item answers once its parent was
// deleted, since a delete cascades. Code 6 means the item exists, but under
// another task. Everything else — including code 1000, an unknown task id,
// whose message is already plain — is returned as is (auth failures get
// their hint at the root).
func describeChecklistDeleteError(err error) error {
	var apiErr *planfix.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.Code {
		case 600:
			return fmt.Errorf("%w — the item does not exist or was already deleted, directly or with a parent item (deleting an item also deletes its nested items)", err)
		case 6:
			return fmt.Errorf("%w — pass the id of the task the item belongs to (see `pfix task checklist list`)", err)
		}
	}
	return err
}
