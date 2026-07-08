// Package groups implements the shared `groups` subcommand that lists the
// groups configured for a resource type (GET /<type>/groups; envelope
// "groups"; columns ID/NAME). It backs `user groups` and `contact groups`.
package groups

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

// NewCmd builds the `groups` subcommand for the given object type ("user" or
// "contact"). objectType is a fixed internal literal, not user input, so it
// needs no validation and is URL-safe by construction.
func NewCmd(g *cmdutil.GlobalOpts, objectType string) *cobra.Command {
	o := &listOptions{objectType: objectType}
	cmd := &cobra.Command{
		Use:   "groups",
		Short: "List " + objectType + " groups",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
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
	fields := cmdutil.FieldsCSV(o.fields, listDefaultFields)
	path := o.objectType + "/groups?fields=" + url.QueryEscape(fields)
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
		Groups []map[string]any `json:"groups"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), env.Groups, !o.quiet)
	return nil
}
