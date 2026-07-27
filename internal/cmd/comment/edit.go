package comment

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type editOptions struct {
	body   map[string]any
	silent bool
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
	in     io.Reader
}

func newEditCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	var body string
	var pinned, hidden, silent bool
	cmd := &cobra.Command{
		Use:   "edit <id>",
		Short: "Edit a comment",
		Long: "Edit a comment.\n\n" +
			"Only the flags you pass are sent, so changing the text leaves the pinned\n" +
			"and hidden state untouched. With no flags at all the new text is read from\n" +
			"stdin. Use --pinned=false / --hidden=false to clear either flag.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			o := &editOptions{
				body:   editBody(body, pinned, hidden, cmd.Flags().Changed),
				silent: silent,
				json:   g.JSON,
				quiet:  g.Quiet,
				jq:     g.JQ,
				client: g.ClientFunc(),
				out:    cmd.OutOrStdout(),
				in:     cmd.InOrStdin(),
			}
			return runEdit(cmd.Context(), o, args[0])
		},
	}
	cmd.Flags().StringVar(&body, "body", "", "New comment text (read from stdin when no flags are given)")
	cmd.Flags().BoolVar(&pinned, "pinned", false, "Pin the comment (--pinned=false to unpin)")
	cmd.Flags().BoolVar(&hidden, "hidden", false, "Hide the comment (--hidden=false to unhide)")
	cmd.Flags().BoolVar(&silent, "silent", false, "Do not notify the comment's recipients")
	return cmd
}

// editBody assembles the update body from the flags the user actually set. The
// API applies partial updates, so an absent key leaves the stored value alone.
func editBody(body string, pinned, hidden bool, changed func(string) bool) map[string]any {
	b := map[string]any{}
	if changed("body") {
		b["description"] = body
	}
	if changed("pinned") {
		b["isPinned"] = pinned
	}
	if changed("hidden") {
		b["isHidden"] = hidden
	}
	return b
}

func runEdit(ctx context.Context, o *editOptions, idStr string) error {
	id, err := cmdutil.ValidateID(idStr)
	if err != nil {
		return err
	}
	if len(o.body) == 0 {
		text, err := io.ReadAll(o.in)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		trimmed := strings.TrimRight(string(text), "\r\n \t")
		if trimmed == "" {
			return fmt.Errorf("nothing to change: pass --body, --pinned or --hidden (or pipe the new text via stdin)")
		}
		o.body = map[string]any{"description": trimmed}
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	parent, err := resolveParent(ctx, client, id)
	if err != nil {
		return err
	}
	path := parent + "/comments/" + strconv.Itoa(id)
	if o.silent {
		path += "?silent=true"
	}
	raw, err := client.JSON(ctx, "POST", path, o.body)
	if err != nil {
		return cmdutil.DescribeAPIError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", id)
		return nil
	}
	fmt.Fprintf(o.out, "Updated comment %d\n", id)
	return nil
}

// resolveParent reports which object a comment hangs off, as that object's path
// prefix ("task/12" or "contact/contact:3").
//
// The update endpoint takes a parent segment but ignores it — posting a task
// comment through a contact's path, or through a task id that does not exist,
// both succeed and neither re-parents the comment. pfix resolves the real
// parent anyway: depending on that laxness would break every edit the day the
// API starts validating it, and this lookup doubles as the existence check for
// an unknown or already-deleted comment.
func resolveParent(ctx context.Context, client *planfix.Client, id int) (string, error) {
	raw, err := client.JSON(ctx, "GET", "comment/"+strconv.Itoa(id)+"?fields=task,contact", nil)
	if err != nil {
		return "", cmdutil.DescribeAPIError(err)
	}
	var env struct {
		Comment map[string]any `json:"comment"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return "", err
	}
	// A task comment carries task (and possibly project); a contact comment
	// carries contact, whose id arrives prefixed ("contact:3"). The endpoint's
	// path accepts that prefixed form, so it is forwarded verbatim.
	for _, key := range []string{"task", "contact"} {
		parent, ok := env.Comment[key].(map[string]any)
		if !ok {
			continue
		}
		if s := output.Flatten(parent, "id"); s != "" {
			return key + "/" + s, nil
		}
	}
	return "", fmt.Errorf("comment %d has no task or contact to edit it through", id)
}
