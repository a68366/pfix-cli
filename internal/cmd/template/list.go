package template

import (
	"context"
	"io"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const listDefaultFields = "id,name"

var listColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
}

type listOptions struct {
	objectType string
	json       bool
	fields     string
	quiet      bool
	jq         string
	client     func() (*planfix.Client, error)
	out        io.Writer
}

func newListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &listOptions{}
	cmd := &cobra.Command{
		Use:   "list <type>",
		Short: "List templates for an object type (task, project, contact, ...)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.objectType = args[0]
			o.json, o.fields, o.quiet = g.JSON, g.Fields, g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runList(cmd.Context(), o)
		},
	}
	return cmd
}

func runList(ctx context.Context, o *listOptions) error {
	if err := cmdutil.ValidateObjectType(o.objectType); err != nil {
		return err
	}
	fields := cmdutil.FieldsCSV(o.fields, listDefaultFields)
	path := o.objectType + "/templates?fields=" + url.QueryEscape(fields)
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
		Templates []map[string]any `json:"templates"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), env.Templates, !o.quiet)
	return nil
}
