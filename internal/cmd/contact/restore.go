package contact

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type restoreOptions struct {
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newRestoreCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &restoreOptions{}
	cmd := &cobra.Command{
		Use:   "restore <id>",
		Short: "Restore a deleted contact",
		Long: "Restore a contact deleted with 'pfix contact delete'. It comes back\n" +
			"intact, under the same id. Restoring a contact that is not deleted fails.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runRestore(cmd.Context(), o, args[0])
		},
	}
	return cmd
}

func runRestore(ctx context.Context, o *restoreOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	// One attempt only: a retry after a restore the server already carried
	// out would answer "not deleted" and misreport the outcome.
	raw, err := client.JSONOnce(ctx, http.MethodPost, contactRestoreEndpoint, map[string]any{"id": id})
	if err == nil {
		err = cmdutil.CheckResult(raw)
	}
	if err != nil {
		return describeContactError(err, true)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", id)
		return nil
	}
	fmt.Fprintf(o.out, "Restored contact %d\n", id)
	return nil
}
