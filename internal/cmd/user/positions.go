package user

import (
	"context"
	"io"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const positionsDefaultFields = "id,name"

var positionsColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
}

type positionsOptions struct {
	json   bool
	fields string
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newPositionsCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &positionsOptions{}
	cmd := &cobra.Command{
		Use:   "positions",
		Short: "List user job positions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json, o.fields, o.quiet = g.JSON, g.Fields, g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runPositions(cmd.Context(), o)
		},
	}
	return cmd
}

func runPositions(ctx context.Context, o *positionsOptions) error {
	fields := cmdutil.FieldsCSV(o.fields, positionsDefaultFields)
	path := "user/positions?fields=" + url.QueryEscape(fields)
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
		Positions []map[string]any `json:"positions"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, positionsDefaultFields, positionsColumns), env.Positions, !o.quiet)
	return nil
}
