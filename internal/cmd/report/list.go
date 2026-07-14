package report

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const listDefaultFields = "id,name"

const listAvailableFields = "id,name,fields"

var listColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
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
		Short: "List reports",
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
	cmd.Flags().IntVar(&o.limit, "limit", 100, "Maximum reports to return")
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
	raw, err := client.JSON(ctx, "POST", "report/list", body)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Reports []map[string]any `json:"reports"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), env.Reports, !o.quiet)
	return nil
}
