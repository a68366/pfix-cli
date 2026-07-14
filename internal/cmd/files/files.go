// Package files implements the shared `files` subcommand that lists the files
// on a task, contact, or project. Attached files come from GET
// /<type>/{id}/files; inline editor images (which that endpoint never returns)
// are scraped from description/comment HTML with --source inline. It backs
// `task files`, `contact files`, and `project files`.
package files

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"

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

func runInline(ctx context.Context, o *listOptions) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	html, err := gatherInlineHTML(ctx, client, o)
	if err != nil {
		return err
	}
	ids := cmdutil.ScanFileIDs(html)
	if len(ids) > 50 && !o.quiet {
		fmt.Fprintf(o.errOut, "resolving %d inline files…\n", len(ids))
	}
	filesList := make([]map[string]any, 0, len(ids))
	for _, id := range ids {
		raw, err := client.JSON(ctx, "GET", "file/"+strconv.Itoa(id), nil)
		if err != nil {
			return cmdutil.DescribeAPIError(err)
		}
		var env struct {
			File map[string]any `json:"file"`
		}
		if err := cmdutil.DecodeJSON(raw, &env); err != nil {
			return err
		}
		if env.File != nil {
			filesList = append(filesList, env.File)
		}
	}
	if o.json {
		composed, err := json.Marshal(map[string]any{"result": "success", "files": filesList})
		if err != nil {
			return err
		}
		return output.EmitJSON(o.out, composed, o.jq)
	}
	renderFiles(o, filesList, "no inline files")
	return nil
}

// gatherInlineHTML returns the HTML to scan for inline file ids, using the
// strategy that fits where the resource keeps its HTML (spec §Where the HTML
// lives): a task's description is comment #1, so its comment feed covers it; a
// contact's description field is plaintext, so only its comments carry HTML; a
// project has no comments, so its (HTML) description is read directly.
func gatherInlineHTML(ctx context.Context, client *planfix.Client, o *listOptions) (string, error) {
	switch o.Type {
	case "task", "contact":
		return gatherCommentHTML(ctx, client, o.Type, o.id)
	case "project":
		raw, err := client.JSON(ctx, "GET", "project/"+strconv.Itoa(o.id)+"?fields=description", nil)
		if err != nil {
			return "", cmdutil.DescribeAPIError(err)
		}
		var env struct {
			Project struct {
				Description string `json:"description"`
			} `json:"project"`
		}
		if err := cmdutil.DecodeJSON(raw, &env); err != nil {
			return "", err
		}
		return env.Project.Description, nil
	default:
		return "", fmt.Errorf("inline files not supported for %s", o.Type)
	}
}

// gatherCommentHTML pages a resource's comment feed and concatenates every
// comment body. Scanning the concatenation deduplicates ids across pages via
// ScanFileIDs. Stops on the first short (<100) page.
func gatherCommentHTML(ctx context.Context, client *planfix.Client, typ string, id int) (string, error) {
	var b strings.Builder
	path := typ + "/" + strconv.Itoa(id) + "/comments/list"
	for offset := 0; ; offset += 100 {
		body := map[string]any{"offset": offset, "pageSize": 100, "fields": "description"}
		raw, err := client.JSON(ctx, "POST", path, body)
		if err != nil {
			return "", cmdutil.DescribeAPIError(err)
		}
		var env struct {
			Comments []struct {
				Description string `json:"description"`
			} `json:"comments"`
		}
		if err := cmdutil.DecodeJSON(raw, &env); err != nil {
			return "", err
		}
		for _, c := range env.Comments {
			b.WriteString(c.Description)
			b.WriteByte('\n')
		}
		if len(env.Comments) < 100 {
			break
		}
	}
	return b.String(), nil
}
