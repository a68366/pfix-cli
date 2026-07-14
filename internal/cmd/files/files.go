// Package files implements the shared `files` subcommand that lists the files
// on a task, contact, or project. Attached files come from GET
// /<type>/{id}/files; inline editor images (which that endpoint never returns)
// are scraped from description/comment HTML with --source inline. It backs
// `task files`, `contact files`, and `project files`.
package files

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

const listDefaultFields = "id,name,size"

var listColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "NAME", Path: "name"},
	{Header: "SIZE", Path: "size"},
}

// Options configures which flags the files command exposes for a parent
// resource. Type is a fixed internal literal ("task"/"contact"/"project"), not
// user input, so it is URL-safe by construction.
type Options struct {
	Type            string
	Paging          bool
	DescriptionOnly bool
}

type listOptions struct {
	Options
	id              int
	source          string
	descriptionOnly bool
	limit, offset   int
	pagingSet       bool
	json            bool
	fields          string
	quiet           bool
	jq              string
	client          func() (*planfix.Client, error)
	out             io.Writer
	errOut          io.Writer
}

// NewCmd builds the `files` subcommand for a parent resource.
func NewCmd(g *cmdutil.GlobalOpts, opts Options) *cobra.Command {
	o := &listOptions{Options: opts}
	cmd := &cobra.Command{
		Use:   "files <id>",
		Short: "List files on a " + opts.Type,
		Long: "List files on a " + opts.Type + ".\n\n" +
			"--source attached (default) lists attachment records. --source inline lists\n" +
			"editor images embedded in the description/comments, which the attachment API\n" +
			"never returns. --fields selects table columns only; the file endpoints do not\n" +
			"support server-side field selection.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			o.id = id
			o.json, o.fields, o.quiet, o.jq = g.JSON, g.Fields, g.Quiet, g.JQ
			o.pagingSet = cmd.Flags().Changed("limit") || cmd.Flags().Changed("offset")
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			o.errOut = cmd.ErrOrStderr()
			return runFiles(cmd.Context(), o)
		},
	}
	f := cmd.Flags()
	f.StringVar(&o.source, "source", "attached", "Which files to list: attached|inline")
	if opts.DescriptionOnly {
		f.BoolVar(&o.descriptionOnly, "description-only", false, "Only files on the object description (attached only)")
	}
	if opts.Paging {
		f.IntVar(&o.limit, "limit", 100, "Maximum files to return")
		f.IntVar(&o.offset, "offset", 0, "Result offset (for paging)")
	}
	return cmd
}

func runFiles(ctx context.Context, o *listOptions) error {
	switch o.source {
	case "attached":
		return runAttached(ctx, o)
	case "inline":
		if o.descriptionOnly {
			return fmt.Errorf("--description-only cannot be combined with --source inline")
		}
		if o.pagingSet {
			return fmt.Errorf("--limit/--offset cannot be combined with --source inline")
		}
		return runInline(ctx, o)
	default:
		return fmt.Errorf("invalid --source %q: use attached or inline", o.source)
	}
}

func runAttached(ctx context.Context, o *listOptions) error {
	q := url.Values{}
	if o.Paging {
		q.Set("pageSize", strconv.Itoa(o.limit))
		q.Set("offset", strconv.Itoa(o.offset))
	}
	if o.descriptionOnly {
		q.Set("onlyFromDescription", "true")
	}
	path := o.Type + "/" + strconv.Itoa(o.id) + "/files"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "GET", path, nil)
	if err != nil {
		return cmdutil.DescribeAPIError(err)
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	var env struct {
		Files []map[string]any `json:"files"`
	}
	if err := cmdutil.DecodeJSON(raw, &env); err != nil {
		return err
	}
	renderFiles(o, env.Files, "no files")
	return nil
}

// renderFiles prints a note to stderr when the set is empty (unless quiet), then
// the aligned table.
func renderFiles(o *listOptions, files []map[string]any, emptyNote string) {
	if len(files) == 0 && !o.quiet {
		fmt.Fprintln(o.errOut, emptyNote)
	}
	fields := cmdutil.FieldsCSV(o.fields, listDefaultFields)
	output.Table(o.out, output.ColumnsFor(fields, listDefaultFields, listColumns), files, !o.quiet)
}

// runInline is implemented in Task 8; stubbed so attached mode ships first.
func runInline(ctx context.Context, o *listOptions) error {
	return fmt.Errorf("--source inline is not implemented yet")
}
