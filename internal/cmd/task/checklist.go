package task

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const checklistListFields = "id,name,isDone"

// checklistViewFields requests assignees too: the people field has no detail
// row (it is a users/groups object, which flattens to nothing) but it enriches
// --json output.
const checklistViewFields = "id,name,isDone,dateTime,assignees"

// checklistAvailableFields is the checklist item vocabulary. Unusually for this
// API, the endpoints enumerate it, so the list is exact rather than advisory.
const checklistAvailableFields = "id,name,isDone,parent,dateTime,assignees"

var checklistColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "DONE", Path: "isDone"},
	{Header: "NAME", Path: "name"},
}

var checklistViewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "DONE", Path: "isDone"},
	{Header: "CREATED", Path: "dateTime.datetime"},
}

// newChecklistCmd returns the `checklist` sub-group. Checklists hang off tasks
// only — no other object type has them — so the whole group is task-scoped.
func newChecklistCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "checklist",
		Short: "Work with a task's checklist",
		Long: "Work with a task's checklist.\n\n" +
			"Items can be listed, viewed, added and updated. The API exposes no\n" +
			"delete, so an item added by mistake can only be renamed, not removed.",
	}
	cmd.AddCommand(newChecklistListCmd(g), newChecklistViewCmd(g), newChecklistAddCmd(g), newChecklistUpdateCmd(g))
	return cmd
}

// checklistItemPath is the route for one item. The API resolves an item by its
// own id and does not check it against the task segment — an item answers
// through any task id, including one that does not exist — so pfix forwards the
// task id the user gave rather than deriving one, and never treats the segment
// as proof of ownership.
func checklistItemPath(taskID, itemID int) string {
	return "task/" + strconv.Itoa(taskID) + "/checklist/" + strconv.Itoa(itemID)
}

// --- checklist list ---

type checklistListOptions struct {
	taskID int
	limit  int
	offset int
	fields string
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
	errOut io.Writer
}

func newChecklistListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &checklistListOptions{}
	cmd := &cobra.Command{
		Use:   "list <task-id>",
		Short: "List a task's checklist items",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			o.taskID = id
			o.fields, o.json, o.quiet, o.jq = g.Fields, g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			o.errOut = cmd.ErrOrStderr()
			return runChecklistList(cmd.Context(), o)
		},
	}
	cmd.Flags().IntVar(&o.limit, "limit", 100, "Maximum items to return (API maximum: 100)")
	cmd.Flags().IntVar(&o.offset, "offset", 0, "Result offset (for paging)")
	cmd.Long = cmdutil.FieldsHelp(cmd.Short, checklistListFields, checklistAvailableFields, "")
	return cmd
}

func runChecklistList(ctx context.Context, o *checklistListOptions) error {
	fields := cmdutil.FieldsCSV(o.fields, checklistListFields)
	body := map[string]any{
		"offset":   o.offset,
		"pageSize": o.limit,
		"fields":   fields,
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	path := "task/" + strconv.Itoa(o.taskID) + "/checklist/list"
	raw, err := client.JSON(ctx, "POST", path, body)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Items []map[string]any `json:"items"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	if len(env.Items) == 0 && !o.quiet {
		fmt.Fprintf(o.errOut, "pfix: task %d has no checklist items\n", o.taskID)
	}
	output.Table(o.out, output.ColumnsFor(fields, checklistListFields, checklistColumns), env.Items, !o.quiet)
	return nil
}

// --- checklist view ---

type checklistViewOptions struct {
	taskID int
	itemID int
	fields string
	json   bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newChecklistViewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &checklistViewOptions{}
	cmd := &cobra.Command{
		Use:   "view <task-id> <item-id>",
		Short: "View one checklist item",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			itemID, err := cmdutil.ValidateID(args[1])
			if err != nil {
				return err
			}
			o.taskID, o.itemID = taskID, itemID
			o.fields, o.json, o.jq = g.Fields, g.JSON, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runChecklistView(cmd.Context(), o)
		},
	}
	cmd.Long = cmdutil.FieldsHelp(cmd.Short, checklistViewFields, checklistAvailableFields, "") +
		"\n\nThe API resolves an item by its own id: the task id is required by the\n" +
		"route but is not checked against the item, so a wrong one still returns\n" +
		"the item. Take the pair from 'pfix task checklist list <task-id>'."
	return cmd
}

func runChecklistView(ctx context.Context, o *checklistViewOptions) error {
	fields := cmdutil.FieldsCSV(o.fields, checklistViewFields)
	client, err := o.client()
	if err != nil {
		return err
	}
	path := checklistItemPath(o.taskID, o.itemID) + "?fields=" + url.QueryEscape(fields)
	raw, err := client.JSON(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Item map[string]any `json:"item"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Detail(o.out, output.ColumnsFor(fields, checklistViewFields, checklistViewColumns), env.Item)
	return nil
}
