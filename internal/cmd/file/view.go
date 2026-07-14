package file

import (
	"context"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

var viewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	// SIZE is the API's approximate kilobytes (ceil(bytes/1024)), not exact bytes.
	{Header: "SIZE", Path: "size"},
	{Header: "LINK", Path: "link"},
}

type viewOptions struct {
	json   bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newViewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View file metadata (id, name, size)",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.jq = g.JSON, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runView(cmd.Context(), o, args[0])
		},
	}
	return cmd
}

func runView(ctx context.Context, o *viewOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "GET", "file/"+strconv.Itoa(id), nil)
	if err != nil {
		return cmdutil.DescribeAPIError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		File map[string]any `json:"file"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	cols := viewColumns
	if _, ok := env.File["link"]; !ok {
		cols = viewColumns[:3] // drop LINK when absent
	}
	output.Detail(o.out, cols, env.File)
	return nil
}
