package contact

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const listDefaultFields = "id,name,lastname,email,isCompany"

var listColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "LASTNAME", Path: "lastname"},
	{Header: "EMAIL", Path: "email"},
	{Header: "COMPANY", Path: "isCompany"},
}

type listOptions struct {
	limit, offset int
	json          bool
	fields        string
	quiet         bool
	filter        string
	client        func() (*planfix.Client, error)
	out           io.Writer
}

func newListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List contacts",
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
	cmd.Flags().IntVar(&o.limit, "limit", 100, "Maximum contacts to return")
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
	raw, err := client.JSON(ctx, "POST", "contact/list", body)
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}
	var env struct {
		Contacts []map[string]any `json:"contacts"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), env.Contacts, !o.quiet)
	return nil
}
