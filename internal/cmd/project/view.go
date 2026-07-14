package project

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

// viewDefaultFields requests assignees too: it has no detail column (the
// {users,groups} shape renders poorly in a flat cell) but enriches `--json`.
const viewDefaultFields = "id,name,description,owner,status,assignees"

const viewAvailableFields = "id,name,description,status,owner,parent,template,group,counterparty,startDate,endDate,hiddenForEmployees,hiddenForClients,overdue,isCloseToDeadline,hasEndDate,assignees,participants,auditors,clientManagers,isDeleted,sourceObjectId,sourceDataVersion"

var viewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "DESCRIPTION", Path: "description"},
	{Header: "OWNER", Path: "owner.name"},
	{Header: "STATUS", Path: "status.id"},
}

type viewOptions struct {
	json   bool
	fields string
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newViewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json = g.JSON
			o.fields = g.Fields
			o.quiet = g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runView(cmd.Context(), o, args[0])
		},
	}
	cmd.Long = cmdutil.FieldsHelp(cmd.Short, viewDefaultFields, viewAvailableFields, "")
	return cmd
}

func runView(ctx context.Context, o *viewOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	fields := cmdutil.FieldsCSV(o.fields, viewDefaultFields)
	path := "project/" + strconv.Itoa(id) + "?fields=" + url.QueryEscape(fields)
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
		Project map[string]any `json:"project"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Detail(o.out, output.ColumnsFor(fields, viewDefaultFields, viewColumns), env.Project)
	return nil
}
