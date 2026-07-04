package task

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const listDefaultFields = "id,name,status,priority,dateTime"

var listColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "STATUS", Path: "status.name"},
	{Header: "PRIORITY", Path: "priority"},
	{Header: "CREATED", Path: "dateTime.datetime"},
}

type listOptions struct {
	limit, offset int
	json          bool
	fields        string
	quiet         bool
	filter        string
	savedFilter   string
	client        func() (*planfix.Client, error)
	out           io.Writer
}

func newListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.fields = g.Fields
			o.quiet = g.Quiet
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runList(cmd.Context(), o)
		},
	}
	cmd.Flags().IntVar(&o.limit, "limit", 100, "Maximum tasks to return")
	cmd.Flags().IntVar(&o.offset, "offset", 0, "Result offset (for paging)")
	cmd.Flags().StringVar(&o.filter, "filter", "", "Filter results: a Planfix filters JSON array, e.g. '[{\"type\":51,\"operator\":\"equal\",\"value\":1}]'")
	cmd.Flags().StringVar(&o.savedFilter, "saved-filter", "", "Apply a saved task filter by id (from 'task filters'), e.g. :in or 220612")
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
	if o.savedFilter != "" {
		body["filterId"] = o.savedFilter
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", "task/list", body)
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}
	var env struct {
		Tasks []map[string]any `json:"tasks"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), env.Tasks, !o.quiet)
	return nil
}
