package object

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const listDefaultFields = "id,name,status,priority"

const listAvailableFields = "id,name,description,priority,status,resultChecking,assigner,parent,project,counterparty,dateTime,startDateTime,endDateTime,hasStartDate,hasEndDate,hasStartTime,hasEndTime,dateOfLastUpdate,duration,durationUnit,durationType,inFavorites,isSummary,isSequential,assignees,participants,auditors,customFieldData,isDeleted,files,sourceObjectId,sourceDataVersion"

var listColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "STATUS", Path: "status.name"},
	{Header: "PRIORITY", Path: "priority"},
}

type listOptions struct {
	limit, offset int
	json          bool
	fields        string
	quiet         bool
	filter        string
	jq            string
	client        func() (*planfix.Client, error)
	out           io.Writer
}

func newListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List objects",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.fields = g.Fields
			o.quiet = g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runList(cmd.Context(), o)
		},
	}
	cmd.Long = cmdutil.FieldsHelp(cmd.Short, listDefaultFields, listAvailableFields, "")
	cmd.Flags().IntVar(&o.limit, "limit", 100, "Maximum objects to return")
	cmd.Flags().IntVar(&o.offset, "offset", 0, "Result offset (for paging)")
	cmd.Flags().StringVar(&o.filter, "filter", "", "Filter results: a Planfix filters JSON array, e.g. '[{\"type\":51,\"operator\":\"equal\",\"value\":1}]'")
	return cmd
}

func runList(ctx context.Context, o *listOptions) error {
	fields := cmdutil.FieldsCSV(o.fields, listDefaultFields)
	body := map[string]any{
		"offset":   o.offset,
		"pageSize": o.limit,
		"fields":   fields,
	}
	if err := cmdutil.ApplyFilter(body, o.filter); err != nil {
		return err
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", "object/list", body)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Objects []map[string]any `json:"objects"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), env.Objects, !o.quiet)
	return nil
}
