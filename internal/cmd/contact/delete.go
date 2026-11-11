package contact

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// Contact app codes seen on delete and restore.
const (
	codeContactNotFound    = 2000 // unknown id, or (on delete) already deleted
	codeContactNotDeleted  = 2002 // restore of a contact that is not deleted
	contactRestoreEndpoint = "contact/restore/"
)

type deleteOptions struct {
	force  bool
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newDeleteCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &deleteOptions{}
	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a contact (restorable)",
		Long: "Delete a contact.\n\n" +
			"Deleting requires --force. The contact moves to the trash: it disappears\n" +
			"from 'pfix contact view' and 'pfix contact list', and comes back intact,\n" +
			"under the same id, with 'pfix contact restore <id>'.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runDelete(cmd.Context(), o, args[0])
		},
	}
	cmd.Flags().BoolVar(&o.force, "force", false, "Confirm the deletion (required)")
	return cmd
}

func runDelete(ctx context.Context, o *deleteOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	if !o.force {
		return fmt.Errorf("refusing to delete contact %d without --force", id)
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	// One attempt only: a retry after a delete the server already carried
	// out would answer "not found" and misreport the outcome.
	raw, err := client.JSONOnce(ctx, http.MethodDelete, "contact/"+strconv.Itoa(id), nil)
	if err == nil {
		err = cmdutil.CheckResult(raw)
	}
	if err != nil {
		return describeContactError(err, false)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", id)
		return nil
	}
	fmt.Fprintf(o.out, "Deleted contact %d (restore with: pfix contact restore %d)\n", id, id)
	return nil
}

// describeContactError adds a hint for the contact app codes whose message
// alone can mislead; everything else is returned as is (auth failures get
// their hint at the root). On delete,
// code 2000 also covers a contact already in the trash; on restore it does
// not (a trashed contact restores), so the hint differs.
func describeContactError(err error, restore bool) error {
	var apiErr *planfix.APIError
	if errors.As(err, &apiErr) {
		switch {
		case apiErr.Code == codeContactNotFound && restore:
			return fmt.Errorf("%w — there is no contact with that id", err)
		case apiErr.Code == codeContactNotFound:
			return fmt.Errorf("%w — there is no contact with that id, or it was already deleted (restore with `pfix contact restore`)", err)
		case apiErr.Code == codeContactNotDeleted && restore:
			return fmt.Errorf("%w — the contact is not deleted, so there is nothing to restore", err)
		}
	}
	return err
}
