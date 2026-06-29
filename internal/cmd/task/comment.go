package task

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

const commentListFields = "id,description,dateTime"

var commentColumns = []output.Column{
	{Header: "ID", Path: "id"},
	{Header: "CREATED", Path: "dateTime.datetime"},
	{Header: "COMMENT", Path: "description"},
}

// newCommentCmd returns the `comment` sub-group with `list` and `add` subcommands.
func newCommentCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "comment",
		Short: "Work with task comments",
	}
	cmd.AddCommand(newCommentListCmd(g), newCommentAddCmd(g))
	return cmd
}

// --- comment list ---

type commentListOptions struct {
	id     int
	limit  int
	offset int
	json   bool
	quiet  bool
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newCommentListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &commentListOptions{}
	cmd := &cobra.Command{
		Use:   "list <id>",
		Short: "List comments on a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validateID(args[0])
			if err != nil {
				return err
			}
			o.id = id
			o.json = g.JSON
			o.quiet = g.Quiet
			o.client = clientFunc(g)
			o.out = cmd.OutOrStdout()
			return runCommentList(cmd.Context(), o)
		},
	}
	cmd.Flags().IntVar(&o.limit, "limit", 100, "Maximum comments to return")
	cmd.Flags().IntVar(&o.offset, "offset", 0, "Result offset (for paging)")
	return cmd
}

func runCommentList(ctx context.Context, o *commentListOptions) error {
	body := map[string]any{
		"offset":   o.offset,
		"pageSize": o.limit,
		"fields":   commentListFields,
	}
	client, err := o.client()
	if err != nil {
		return err
	}
	path := "task/" + strconv.Itoa(o.id) + "/comments/list"
	raw, err := client.JSON(ctx, "POST", path, body)
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}
	var env struct {
		Comments []map[string]any `json:"comments"`
	}
	if err := jsonUnmarshal(raw, &env); err != nil {
		return err
	}
	// Truncate description to 80 runes (collapse newlines first).
	for _, c := range env.Comments {
		if d, ok := c["description"].(string); ok {
			d = strings.ReplaceAll(d, "\n", " ")
			c["description"] = output.Truncate(d, 80)
		}
	}
	output.Table(o.out, commentColumns, env.Comments, !o.quiet)
	return nil
}

// --- comment add ---

type commentAddOptions struct {
	id     int
	body   string
	json   bool
	quiet  bool
	client func() (*planfix.Client, error)
	out    io.Writer
	in     io.Reader
}

func newCommentAddCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &commentAddOptions{}
	cmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Add a comment to a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := validateID(args[0])
			if err != nil {
				return err
			}
			o.id = id
			o.json = g.JSON
			o.quiet = g.Quiet
			o.client = clientFunc(g)
			o.out = cmd.OutOrStdout()
			o.in = cmd.InOrStdin()
			return runCommentAdd(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&o.body, "body", "", "Comment body (or pipe via stdin)")
	return cmd
}

func runCommentAdd(ctx context.Context, o *commentAddOptions) error {
	body := o.body
	if body == "" {
		b, err := io.ReadAll(o.in)
		if err != nil {
			return fmt.Errorf("read stdin: %w", err)
		}
		body = strings.TrimRight(string(b), "\r\n \t")
	}
	if body == "" {
		return fmt.Errorf("comment body is required (use --body or pipe via stdin)")
	}

	client, err := o.client()
	if err != nil {
		return err
	}
	path := "task/" + strconv.Itoa(o.id) + "/comments/"
	raw, err := client.JSON(ctx, "POST", path, map[string]any{"description": body})
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}

	var resp struct {
		ID int `json:"id"`
	}
	if err := jsonUnmarshal(raw, &resp); err != nil {
		return err
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", resp.ID)
		return nil
	}
	fmt.Fprintf(o.out, "Added comment %d\n", resp.ID)
	return nil
}
