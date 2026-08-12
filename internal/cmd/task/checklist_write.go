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

// checklistBody assembles an item payload from the flags the user actually set.
// The update endpoint applies partial updates, so an absent key leaves the
// stored value alone — and --done=false has to be distinguishable from an
// unmentioned --done, which is why the caller passes cmd.Flags().Changed.
func checklistBody(name string, done bool, assignees []string, changed func(string) bool) (map[string]any, error) {
	b := map[string]any{}
	if changed("name") {
		b["name"] = name
	}
	if changed("done") {
		b["isDone"] = done
	}
	if changed("assignees") {
		people, err := cmdutil.ParsePeople(assignees)
		if err != nil {
			return nil, err
		}
		b["assignees"] = people
	}
	return b, nil
}

// checklistFailure is one entry of the update endpoint's per-field rejection
// list.
type checklistFailure struct {
	Field string `json:"field"`
	Error string `json:"error"`
}

// checklistFailuresOf reads the failures array out of a write response,
// best-effort: a response without one (create documents no schema at all) or a
// body that does not decode simply reports nothing to complain about.
func checklistFailuresOf(raw []byte) []checklistFailure {
	var resp struct {
		Failures []checklistFailure `json:"failures"`
	}
	_ = json.Unmarshal(raw, &resp)
	return resp.Failures
}

// checklistFailureError turns those entries into an error. The endpoint answers
// 200 with result "success" and names what it refused in failures[], so an
// unchecked write would read as a successful no-op.
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
	var name string
	var done bool
	var assignees []string
	cmd := &cobra.Command{
		Use:   "add <task-id>",
		Short: "Add an item to a task's checklist",
		Long: "Add an item to a task's checklist.\n\n" +
			"The API has no delete, so an item added here can later be renamed or\n" +
			"ticked off, but never removed.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			body, err := checklistBody(name, done, assignees, cmd.Flags().Changed)
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
	cmd.Flags().StringVar(&name, "name", "", "Item text (required)")
	cmd.Flags().BoolVar(&done, "done", false, "Create the item already ticked off")
	cmd.Flags().StringSliceVar(&assignees, "assignees", nil, "Assignees: user:N, contact:N, or group:N (comma-separated)")
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
	// The create response has no documented schema. Report the new id when one
	// comes back and stay quiet about it when it does not, rather than printing
	// a zero that looks like an item number.
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
	var name string
	var done bool
	var assignees []string
	cmd := &cobra.Command{
		Use:   "update <task-id> <item-id>",
		Short: "Update a checklist item",
		Long: "Update a checklist item.\n\n" +
			"Only the flags you pass are sent, so ticking an item leaves its text\n" +
			"and assignees untouched. Use --done=false to untick one.\n\n" +
			"The API resolves an item by its own id: the task id is required by the\n" +
			"route but is not checked against the item, so a wrong one still updates\n" +
			"the item. Take the pair from 'pfix task checklist list <task-id>'.",
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
			body, err := checklistBody(name, done, assignees, cmd.Flags().Changed)
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
	cmd.Flags().StringVar(&name, "name", "", "New item text")
	cmd.Flags().BoolVar(&done, "done", false, "Tick the item off (--done=false to untick)")
	cmd.Flags().StringSliceVar(&assignees, "assignees", nil, "Replace the assignees: user:N, contact:N, or group:N (comma-separated)")
	return cmd
}

func runChecklistUpdate(ctx context.Context, o *checklistUpdateOptions) error {
	if len(o.body) == 0 {
		return fmt.Errorf("nothing to change: pass --name, --done or --assignees")
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
