package task

import (
	"context"
	"fmt"
	"io"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const statusesDefaultFields = "id,name,isActive"

var statusesColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "ACTIVE", Path: "isActive"},
}

type statusesOptions struct {
	taskID  string
	process int
	json    bool
	fields  string
	quiet   bool
	jq      string
	client  func() (*planfix.Client, error)
	out     io.Writer
	errOut  io.Writer
}

// newStatusesCmd returns the `statuses` subcommand: the status set a task's
// --status can use. It resolves a task's processId
// (GET /task/{id}?fields=processId) or takes --process directly, then
// GET /process/task/{processId}/statuses.
func newStatusesCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &statusesOptions{}
	cmd := &cobra.Command{
		Use:   "statuses [task-id]",
		Short: "List the statuses a task's --status can use",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 1 {
				o.taskID = args[0]
			}
			o.json, o.fields, o.quiet = g.JSON, g.Fields, g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			o.errOut = cmd.ErrOrStderr()
			return runStatuses(cmd.Context(), o)
		},
	}
	cmd.Flags().IntVar(&o.process, "process", 0, "Process ID (instead of a task id)")
	return cmd
}

func runStatuses(ctx context.Context, o *statusesOptions) error {
	if o.taskID == "" && o.process == 0 {
		return fmt.Errorf("provide a task id or --process <id>")
	}
	if o.taskID != "" && o.process != 0 {
		return fmt.Errorf("task id and --process are mutually exclusive")
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	processID := o.process
	if o.taskID != "" {
		id, err := cmdutil.ValidateID(o.taskID)
		if err != nil {
			return err
		}
		processID, err = resolveProcessID(ctx, client, id)
		if err != nil {
			return err
		}
	} else if o.process < 0 {
		return fmt.Errorf("--process must be a positive number, got %d", o.process)
	}

	fields := cmdutil.FieldsCSV(o.fields, statusesDefaultFields)
	path := fmt.Sprintf("process/task/%d/statuses?fields=%s", processID, url.QueryEscape(fields))
	raw, err := client.JSON(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Statuses []map[string]any `json:"statuses"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	if len(env.Statuses) == 0 && !o.quiet {
		fmt.Fprintf(o.errOut, "pfix: process %d has no statuses\n", processID)
	}
	output.Table(o.out, output.ColumnsFor(fields, statusesDefaultFields, statusesColumns), env.Statuses, !o.quiet)
	return nil
}

// resolveProcessID reads a task's processId (GET /task/{id}?fields=processId).
func resolveProcessID(ctx context.Context, client *planfix.Client, taskID int) (int, error) {
	raw, err := client.JSON(ctx, "GET", fmt.Sprintf("task/%d?fields=processId", taskID), nil)
	if err != nil {
		return 0, err
	}
	var env struct {
		Task struct {
			ProcessID int `json:"processId"`
		} `json:"task"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return 0, err
	}
	if env.Task.ProcessID <= 0 {
		return 0, fmt.Errorf("could not determine the process for task %d", taskID)
	}
	return env.Task.ProcessID, nil
}
