package user

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

// viewDefaultFields requests position too: no detail column (it's an object)
// but it enriches --json.
const viewDefaultFields = "id,name,midname,lastname,email,login,status,role,position"

var viewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "MIDNAME", Path: "midname"},
	{Header: "LASTNAME", Path: "lastname"},
	{Header: "EMAIL", Path: "email"},
	{Header: "LOGIN", Path: "login"},
	{Header: "STATUS", Path: "status"},
	{Header: "ROLE", Path: "role"},
}

type viewOptions struct {
	json   bool
	fields string
	quiet  bool
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newViewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a user",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json = g.JSON
			o.fields = g.Fields
			o.quiet = g.Quiet
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
	fields := cmdutil.FieldsCSV(o.fields, viewDefaultFields)
	path := "user/" + strconv.Itoa(id) + "?fields=" + url.QueryEscape(fields)
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "GET", path, nil)
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}
	var env struct {
		User map[string]any `json:"user"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Detail(o.out, output.ColumnsFor(fields, viewDefaultFields, viewColumns), env.User)
	return nil
}
