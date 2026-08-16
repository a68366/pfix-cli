package task

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// checklistFlags is the writable item field set, shared by add and update. Each
// command registers its own flags against it so the help text can speak in the
// right tense; only the payload assembly is common.
type checklistFlags struct {
	name      string
	done      bool
	parent    int
	assignees []string
}

// body assembles the item payload from the flags the user actually set. The
// update endpoint applies partial updates, so an absent key leaves the stored
// value alone — and --done=false has to be distinguishable from an unmentioned
// --done, which is why the caller passes cmd.Flags().Changed.
func (f *checklistFlags) body(changed func(string) bool) (map[string]any, error) {
	b := map[string]any{}
	if changed("name") {
		b["name"] = f.name
	}
	if changed("done") {
		b["isDone"] = f.done
	}
	if changed("parent") {
		if f.parent <= 0 {
			return nil, fmt.Errorf("--parent must be a positive checklist item id")
		}
		b["parent"] = map[string]any{"id": f.parent}
	}
	if changed("assignees") {
		// An empty --assignees="" parses as a zero-length slice rather than a
		// flag error, and the list is replaced wholesale, so accepting it would
		// silently clear the item's people — refuse it as taskFields.apply does
		// for the task-level people flags.
		if len(f.assignees) == 0 {
			return nil, fmt.Errorf("--assignees requires at least one reference")
		}
		people, err := cmdutil.ParsePeople(f.assignees)
		if err != nil {
			return nil, err
		}
		b["assignees"] = people
	}
	return b, nil
}

// assigneesHelp warns about the one write the API accepts and then quietly
// discards: a reference it cannot resolve is dropped, and since the list is
// replaced wholesale, the item ends up with no assignees at all — reported as
// success, with nothing in the response to distinguish it from a real change.
const assigneesHelp = " (a reference the API cannot resolve is dropped silently, clearing the list)"

// checklistFailure is one entry of the update endpoint's per-field rejection
// list.
type checklistFailure struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// checklistFailuresOf reads the failures array out of a write response,
// best-effort: a response without one — create documents only {result, id} —
// or a body that does not decode simply reports nothing to complain about.
func checklistFailuresOf(raw []byte) []checklistFailure {
	var resp struct {
		Failures []checklistFailure `json:"failures"`
	}
	_ = json.Unmarshal(raw, &resp)
	return resp.Failures
}

// checklistFailureError turns those entries into an error. The update
// endpoint's documented 200/202 response carries this array alongside
// result "success", so an unchecked write could read as a successful no-op.
// (No probe has provoked a non-empty one — the rejections seen live are either
// a 400 or a silent drop — so this is a documented contract pfix honors rather
// than an observed behavior.)
func checklistFailureError(failures []checklistFailure) error {
	if len(failures) == 0 {
		return nil
	}
	parts := make([]string, 0, len(failures))
	for _, f := range failures {
		if f.Field == "" {
			parts = append(parts, f.Error)
			continue
		}
		parts = append(parts, f.Field+": "+f.Error)
	}
	return fmt.Errorf("the API rejected the request: %s", strings.Join(parts, "; "))
}

// --- checklist add ---

type checklistAddOptions struct {
	taskID int
	body   map[string]any
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newChecklistAddCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &checklistAddOptions{}
	f := &checklistFlags{}
	cmd := &cobra.Command{
		Use:   "add <task-id>",
		Short: "Add an item to a task's checklist",
		Long: "Add an item to a task's checklist.\n\n" +
			"Pass --parent to nest the new item under an existing one; the parent\n" +
			"must belong to the same task. The API has no delete, so an item added\n" +
			"here can later be renamed, ticked off or moved, but never removed.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			body, err := f.body(cmd.Flags().Changed)
			if err != nil {
				return err
			}
			o.taskID, o.body = id, body
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runChecklistAdd(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&f.name, "name", "", "Item text (required)")
	cmd.Flags().BoolVar(&f.done, "done", false, "Create the item already ticked off")
	cmd.Flags().IntVar(&f.parent, "parent", 0, "Nest the item under this checklist item id")
	cmd.Flags().StringSliceVar(&f.assignees, "assignees", nil, "Assignees: user:N, contact:N, or group:N (comma-separated)"+assigneesHelp)
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func runChecklistAdd(ctx context.Context, o *checklistAddOptions) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	path := "task/" + strconv.Itoa(o.taskID) + "/checklist"
	raw, err := client.JSON(ctx, "POST", path, o.body)
	if err != nil {
		return err
	}
	failure := checklistFailureError(checklistFailuresOf(raw))
	if o.json {
		if err := output.EmitJSON(o.out, raw, o.jq); err != nil {
			return err
		}
		return failure
	}
	if failure != nil {
		return failure
	}
	var resp struct {
		ID int `json:"id"`
	}
	if err := cmdutil.DecodeJSON(raw, &resp); err != nil {
		return err
	}
	// Create answers {result, id}. Report the new id when one comes back and
	// stay quiet about it when it does not, rather than printing a zero that
	// looks like an item number.
	if resp.ID <= 0 {
		if o.quiet {
			return nil
		}
		fmt.Fprintf(o.out, "Added checklist item to task %d\n", o.taskID)
		return nil
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", resp.ID)
		return nil
	}
	fmt.Fprintf(o.out, "Added checklist item %d\n", resp.ID)
	return nil
}

// --- checklist update ---

type checklistUpdateOptions struct {
	taskID int
	itemID int
	body   map[string]any
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newChecklistUpdateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &checklistUpdateOptions{}
	f := &checklistFlags{}
	cmd := &cobra.Command{
		Use:   "update <task-id> <item-id>",
		Short: "Update a checklist item",
		Long: "Update a checklist item.\n\n" +
			"Only the flags you pass are sent, so ticking an item leaves its text\n" +
			"and assignees untouched. Use --done=false to untick one, and --parent\n" +
			"to move it under another item of the same task.\n\n" +
			"Unlike the read side, a write checks the pair: the item must belong to\n" +
			"the task id in the route, or the API answers 400.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			taskID, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			itemID, err := cmdutil.ValidateID(args[1])
			if err != nil {
				return err
			}
			body, err := f.body(cmd.Flags().Changed)
			if err != nil {
				return err
			}
			o.taskID, o.itemID, o.body = taskID, itemID, body
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runChecklistUpdate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&f.name, "name", "", "New item text")
	cmd.Flags().BoolVar(&f.done, "done", false, "Tick the item off (--done=false to untick)")
	cmd.Flags().IntVar(&f.parent, "parent", 0, "Move the item under this checklist item id")
	cmd.Flags().StringSliceVar(&f.assignees, "assignees", nil, "Replace the assignees: user:N, contact:N, or group:N (comma-separated)"+assigneesHelp)
	return cmd
}

func runChecklistUpdate(ctx context.Context, o *checklistUpdateOptions) error {
	if len(o.body) == 0 {
		return fmt.Errorf("nothing to change: pass --name, --done, --parent or --assignees")
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", checklistItemPath(o.taskID, o.itemID), o.body)
	if err != nil {
		return err
	}
	failure := checklistFailureError(checklistFailuresOf(raw))
	if o.json {
		if err := output.EmitJSON(o.out, raw, o.jq); err != nil {
			return err
		}
		return failure
	}
	if failure != nil {
		return failure
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", o.itemID)
		return nil
	}
	fmt.Fprintf(o.out, "Updated checklist item %d\n", o.itemID)
	return nil
}
