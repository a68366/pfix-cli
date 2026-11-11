// Package whoami implements `pfix whoami`, which shows the profile behind the
// active token via GET /userinfo, and exposes Fetch/Info for `auth status`.
package whoami

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// columns are the detail rows; a row whose value is empty (e.g. a null email,
// or groups on a contact login, which the API omits) is dropped.
var columns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "LOGIN", Path: "login"},
	{Header: "EMAIL", Path: "email"},
	{Header: "TYPE", Path: "type"},
	{Header: "ACCOUNT", Path: "account_name"},
	{Header: "GROUPS", Path: "groups"},
}

// Info is the subset of the /userinfo response pfix summarizes. The response
// has no "result" envelope: the profile fields sit at the top level.
type Info struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Login string `json:"login"`
}

// Summary renders a compact one-line identity, e.g.
// "Jane Doe (user:1, login jdoe)", omitting whatever the profile lacks.
func (i Info) Summary() string {
	inner := make([]string, 0, 2)
	head := i.ID
	if i.Name != "" {
		head = i.Name
		inner = append(inner, i.ID)
	}
	if i.Login != "" {
		inner = append(inner, "login "+i.Login)
	}
	if len(inner) == 0 {
		return head
	}
	return head + " (" + strings.Join(inner, ", ") + ")"
}

type options struct {
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

// NewCmd builds the top-level `whoami` command.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &options{}
	return &cobra.Command{
		Use:   "whoami",
		Short: "Show the user or contact behind the active token",
		Long: "Show the user or contact behind the active token.\n\n" +
			"Sends GET /userinfo and prints the login's id, name, login, email, type,\n" +
			"account, and groups (groups are present for employee logins only). Empty\n" +
			"values are omitted. -q prints only the id (e.g. user:1); --json prints the\n" +
			"raw response.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.quiet = g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return run(cmd.Context(), o)
		},
	}
}

func run(ctx context.Context, o *options) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := get(ctx, client)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var obj map[string]any
	if err := cmdutil.DecodeJSON(raw, &obj); err != nil {
		return err
	}
	if o.quiet {
		fmt.Fprintln(o.out, output.Flatten(obj, "id"))
		return nil
	}
	cols := make([]output.Column, 0, len(columns))
	for _, c := range columns {
		if output.Flatten(obj, c.Path) != "" {
			cols = append(cols, c)
		}
	}
	output.Detail(o.out, cols, obj)
	return nil
}

// Fetch returns the profile behind the client's token. Its error carries the
// auth hints itself, because callers print it in place rather than return it
// to the root, which adds them for returned errors.
func Fetch(ctx context.Context, client *planfix.Client) (Info, error) {
	raw, err := get(ctx, client)
	if err != nil {
		return Info{}, cmdutil.DescribeAPIError(err, cmdutil.AuthContext{})
	}
	var info Info
	if err := cmdutil.DecodeJSON(raw, &info); err != nil {
		return Info{}, err
	}
	return info, nil
}

func get(ctx context.Context, client *planfix.Client) ([]byte, error) {
	raw, err := client.JSON(ctx, "GET", "userinfo", nil)
	if err != nil {
		return nil, describe(err)
	}
	return raw, nil
}

// describe adds the /userinfo-specific hint: the endpoint answers 403 for a
// login type or profile it does not support. An auth failure is returned as
// is, so the shared auth hint (added by the root, or by Fetch) is the only one.
func describe(err error) error {
	if cmdutil.DescribeAPIError(err, cmdutil.AuthContext{}) != err {
		return err
	}
	var apiErr *planfix.APIError
	if errors.As(err, &apiErr) && apiErr.StatusCode == 403 {
		return fmt.Errorf("%w — the API does not support this login type or profile for user info", err)
	}
	return err
}
