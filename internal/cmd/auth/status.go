package auth

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type statusOptions struct {
	profileName string
	domain      string
	token       string
	client      func() (*planfix.Client, error)
	out         io.Writer
}

func newStatusCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show the active profile and check the token",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, res, err := g.Client()
			if err != nil {
				return err
			}
			o := &statusOptions{
				profileName: res.ProfileName,
				domain:      res.Domain,
				token:       res.Token,
				client:      func() (*planfix.Client, error) { return client, nil },
				out:         cmd.OutOrStdout(),
			}
			return runStatus(cmd.Context(), o)
		},
	}
}

// runStatus prints the active profile and confirms the token by probing
// GET /ping — the lightest call that validates a token independent of any
// resource scope. (A task/list probe would fail for a valid token scoped only
// to, say, contacts, falsely reporting the token as bad.)
func runStatus(ctx context.Context, o *statusOptions) error {
	fmt.Fprintf(o.out, "Profile: %s\n", o.profileName)
	fmt.Fprintf(o.out, "Domain:  %s\n", o.domain)
	fmt.Fprintf(o.out, "Token:   %s\n", cmdutil.MaskToken(o.token))

	client, err := o.client()
	if err != nil {
		return err
	}
	if _, err := client.JSON(ctx, "GET", "ping", nil); err != nil {
		return cmdutil.DescribeAPIError(err)
	}
	fmt.Fprintln(o.out, "Status:  valid")
	return nil
}
