package customfield

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// typeNames maps the Planfix custom-field type code to its display name.
// Source: GET /customfield/type (a stable system catalog). Codes 18 and 19 are
// intentionally absent (the API skips them).
var typeNames = map[int]string{
	0:  "Short text",
	1:  "Number",
	2:  "Multi-line text",
	3:  "Date",
	4:  "Time",
	5:  "Date and time",
	6:  "Period of time",
	7:  "Checkbox",
	8:  "List",
	9:  "Directory entry",
	10: "Contact",
	11: "Employee",
	12: "Counterparty",
	13: "Group, employee, or contact",
	14: "List of users",
	15: "Set of directory values",
	16: "Task",
	17: "Task set",
	20: "Set of values",
	21: "Files",
	22: "Project",
	23: "Data tag summaries",
	24: "Calculated field",
	25: "Location",
	26: "Subtask total",
	27: "AI results field",
	28: "Date with time frame",
	29: "Totals field",
}

// typeName renders a custom-field type code (a float64 from decoded JSON) as its
// catalog name, falling back to the raw number for an unknown code and to a
// plain string for any non-numeric value.
func typeName(v any) string {
	f, ok := v.(float64)
	if !ok {
		return fmt.Sprintf("%v", v)
	}
	code := int(f)
	if n, ok := typeNames[code]; ok {
		return n
	}
	return strconv.Itoa(code)
}

const typesDefaultFields = "id,name"

var typesColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
}

type typesOptions struct {
	json   bool
	fields string
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newTypesCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &typesOptions{}
	cmd := &cobra.Command{
		Use:   "types",
		Short: "List the custom-field type catalog",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json, o.fields, o.quiet = g.JSON, g.Fields, g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runTypes(cmd.Context(), o)
		},
	}
	return cmd
}

func runTypes(ctx context.Context, o *typesOptions) error {
	fields := cmdutil.FieldsCSV(o.fields, typesDefaultFields)
	path := "customfield/type?fields=" + url.QueryEscape(fields)
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
		Types []map[string]any `json:"customFieldTypes"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	output.Table(o.out, output.ColumnsFor(fields, typesDefaultFields, typesColumns), env.Types, !o.quiet)
	return nil
}
