package cmd

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type pingOptions struct {
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newPingCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &pingOptions{}
	cmd := &cobra.Command{
		Use:   "ping",
		Short: "Check REST API connectivity and token validity",
		Long: "Check REST API connectivity and token validity.\n\n" +
			"Sends an authenticated GET /ping. A valid token returns success; an unknown\n" +
			"token fails regardless of its resource scopes, making this the lightest way to\n" +
			"confirm the active profile can reach the API.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.quiet = g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runPing(cmd.Context(), o)
		},
	}
	return cmd
}

func runPing(ctx context.Context, o *pingOptions) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "GET", "ping", nil)
	if err != nil {
		return cmdutil.DescribeAPIError(err)
	}
	if o.quiet {
		return nil
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	fmt.Fprintln(o.out, "OK")
	return nil
}
