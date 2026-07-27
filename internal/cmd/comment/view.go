package comment

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

const viewDefaultFields = "id,dateTime,description,owner,task,contact,isPinned,isHidden,type,fromType"

const viewAvailableFields = "id,dateTime,type,fromType,description,additionalDescriptionData,task,project,contact,owner,isDeleted,isPinned,isHidden,isNotRead,recipients,reminders,dataTags,files,changeTaskStartDate,changeTaskExpectDate,changeStatus,sourceObjectId,sourceDataVersion"

// viewColumns are the detail rows for one comment. TASK and CONTACT are
// mutually exclusive in practice — a comment hangs off one or the other — so
// parentColumns drops whichever the response does not carry.
var viewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "CREATED", Path: "dateTime.datetime"},
	{Header: "AUTHOR", Path: "owner.name"},
	{Header: "TASK", Path: "task.id"},
	{Header: "CONTACT", Path: "contact.id"},
	{Header: "PINNED", Path: "isPinned"},
	{Header: "HIDDEN", Path: "isHidden"},
	{Header: "TEXT", Path: "description"},
}

type viewOptions struct {
	json   bool
	fields string
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newViewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &viewOptions{}
	cmd := &cobra.Command{
		Use:   "view <id>",
		Short: "View a comment",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.fields, o.jq = g.JSON, g.Fields, g.JQ
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
	client, err := o.client()
	if err != nil {
		return err
	}
	path := "comment/" + strconv.Itoa(id) + "?fields=" + url.QueryEscape(fields)
	raw, err := client.JSON(ctx, "GET", path, nil)
	if err != nil {
		return cmdutil.DescribeAPIError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Comment map[string]any `json:"comment"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	cols := output.ColumnsFor(fields, viewDefaultFields, parentColumns(env.Comment))
	output.Detail(o.out, cols, env.Comment)
	return nil
}

// parentColumns drops the parent row the comment does not carry, so a task
// comment never renders an empty CONTACT line and vice versa.
func parentColumns(c map[string]any) []output.Column {
	cols := make([]output.Column, 0, len(viewColumns))
	for _, col := range viewColumns {
		if col.Header == "TASK" && c["task"] == nil {
			continue
		}
		if col.Header == "CONTACT" && c["contact"] == nil {
			continue
		}
		cols = append(cols, col)
	}
	return cols
}
