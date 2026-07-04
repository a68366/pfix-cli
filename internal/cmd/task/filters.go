package task

import (
	"context"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

var filtersColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "OWNER", Path: "owner.name"},
}

type filtersOptions struct {
	json   bool
	quiet  bool
	client func() (*planfix.Client, error)
	out    io.Writer
}

// newFiltersCmd returns the `filters` subcommand: a read-only listing of the
// account's saved task filters (POST /task/filters, no pagination).
func newFiltersCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &filtersOptions{}
	cmd := &cobra.Command{
		Use:   "filters",
		Short: "List saved task filters",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.quiet = g.Quiet
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runFilters(cmd.Context(), o)
		},
	}
	return cmd
}

func runFilters(ctx context.Context, o *filtersOptions) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", "task/filters", map[string]any{})
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}
	var env struct {
		Filters []map[string]any `json:"filters"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, filtersColumns, env.Filters, !o.quiet)
	return nil
}
