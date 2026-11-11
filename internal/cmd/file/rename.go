package file

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type renameOptions struct {
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newRenameCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &renameOptions{}
	return &cobra.Command{
		Use:   "rename <id> <new-name>",
		Short: "Rename a file",
		Long: "Rename a file. The new name shows everywhere the file is attached and is\n" +
			"the name 'pfix file download' saves it under.\n\n" +
			"The name may not be empty or contain a path separator or control\n" +
			"characters: the API stores whatever it is given verbatim, so pfix checks\n" +
			"it first. A deleted file cannot be renamed.",
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			o.json, o.quiet, o.jq = g.JSON, g.Quiet, g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runRename(cmd.Context(), o, args[0], args[1])
		},
	}
}

func runRename(ctx context.Context, o *renameOptions, idStr, name string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	if err := validateUploadName(name, "new name"); err != nil {
		return err
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	// Setting a name is idempotent, so the retrying path is safe: a repeat of
	// a rename the server already applied just sets the same name again.
	raw, err := client.JSON(ctx, http.MethodPost, "file/"+strconv.Itoa(id), map[string]string{"name": name})
	if err == nil {
		// A deleted file answers HTTP 200 with result "fail".
		err = cmdutil.CheckResult(raw)
	}
	if err != nil {
		return describeFileWriteError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", id)
		return nil
	}
	fmt.Fprintf(o.out, "Renamed file %d to %s\n", id, name)
	return nil
}
