package contact

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

// viewDefaultFields requests phones too: no detail column (array renders poorly
// in a flat cell) but it enriches --json.
const viewDefaultFields = "id,name,midname,lastname,email,phones,isCompany,position,description"

const viewAvailableFields = "id,template,name,midname,lastname,gender,description,address,site,email,additionalEmailAddresses,skype,telegramId,telegram,facebook,instagram,viberId,position,group,isCompany,isDeleted,birthDate,createdDate,dateOfLastUpdate,supervisors,phones,companies,contacts,files,dataTags,avatarUrl,addedBy,languageCode,communicationLanguageCode,sourceObjectId,sourceDataVersion"

var viewColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "MIDNAME", Path: "midname"},
	{Header: "LASTNAME", Path: "lastname"},
	{Header: "EMAIL", Path: "email"},
	{Header: "COMPANY", Path: "isCompany"},
	{Header: "POSITION", Path: "position"},
	{Header: "DESCRIPTION", Path: "description"},
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
		Short: "View a contact",
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
	path := "contact/" + strconv.Itoa(id) + "?fields=" + url.QueryEscape(fields)
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
		Contact map[string]any `json:"contact"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Detail(o.out, output.ColumnsFor(fields, viewDefaultFields, viewColumns), env.Contact)
	return nil
}
