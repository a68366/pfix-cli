package task

import (
	"context"
	"io"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const viewDefaultFields = "id,name,description,status,priority,type,dateTime,overdue"

const viewAvailableFields = "id,name,description,additionalDescriptionData,priority,status,processId,resultChecking,type,assigner,parent,object,template,project,counterparty,dateTime,startDateTime,endDateTime,hasStartDate,hasEndDate,hasStartTime,hasEndTime,delayedTillDate,actualCompletionDate,dateOfLastUpdate,dateOfLastComment,duration,durationUnit,durationType,overdue,closeToDeadLine,notAcceptedInTime,inFavorites,isSummary,isSequential,assignees,participants,auditors,projectAuditors,recurrence,isDeleted,files,dataTags,sourceObjectId,sourceDataVersion"

var viewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "DESCRIPTION", Path: "description"},
	{Header: "STATUS", Path: "status.name"},
	{Header: "PRIORITY", Path: "priority"},
	{Header: "TYPE", Path: "type"},
	{Header: "CREATED", Path: "dateTime.datetime"},
	{Header: "OVERDUE", Path: "overdue"},
}

type viewOptions struct {
	json   bool
	fields string
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newViewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json = g.JSON
			o.fields = g.Fields
			o.quiet = g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runView(cmd.Context(), o, args[0])
		},
	}
	cmd.Long = cmdutil.FieldsHelp(cmd.Short, viewDefaultFields, viewAvailableFields, "task")
	return cmd
}

func runView(ctx context.Context, o *viewOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	fields := cmdutil.FieldsCSV(o.fields, viewDefaultFields)
	path := "task/" + strconv.Itoa(id) + "?fields=" + url.QueryEscape(fields)
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Task map[string]any `json:"task"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	cols := dropNumericColumns(output.ColumnsFor(fields, viewDefaultFields, viewColumns))
	output.Detail(o.out, cols, env.Task, customFieldRows(env.Task)...)
	return nil
}

// customFieldRows extracts label/value rows from a task's customFieldData array
// (present only when numeric field ids were requested via --fields). Each entry
// renders as "field.name = stringValue"; entries without a name are skipped.
func customFieldRows(task map[string]any) []output.KV {
	arr, ok := task["customFieldData"].([]any)
	if !ok {
		return nil
	}
	rows := make([]output.KV, 0, len(arr))
	for _, e := range arr {
		entry, ok := e.(map[string]any)
		if !ok {
			continue
		}
		name := ""
		if field, ok := entry["field"].(map[string]any); ok {
			if n, ok := field["name"].(string); ok {
				name = n
			}
		}
		if name == "" {
			continue
		}
		value := ""
		if sv, ok := entry["stringValue"].(string); ok {
			value = sv
		}
		rows = append(rows, output.KV{Key: name, Value: value})
	}
	return rows
}

// dropNumericColumns removes columns whose path is a bare custom-field id (all
// digits) — those values render via customFieldRows, not as a top-level column.
func dropNumericColumns(cols []output.Column) []output.Column {
	out := make([]output.Column, 0, len(cols))
	for _, c := range cols {
		if isAllDigits(c.Path) {
			continue
		}
		out = append(out, c)
	}
	return out
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
