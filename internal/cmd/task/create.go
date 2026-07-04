package task

import (
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type createOptions struct {
	body   map[string]any
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newCreateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	var name, description string
	f := &taskFields{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a task",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			body, err := createBody(name, description, f, cmd.Flags().Changed)
			if err != nil {
				return err
			}
			o := &createOptions{
				body:   body,
				json:   g.JSON,
				quiet:  g.Quiet,
				jq:     g.JQ,
				client: g.ClientFunc(),
				out:    cmd.OutOrStdout(),
			}
			return runCreate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name (required)")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	f.register(cmd, true)
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

// createBody assembles the create request body: the required name, the
// description when non-empty, and every field flag reported set by changed.
func createBody(name, description string, f *taskFields, changed func(string) bool) (map[string]any, error) {
	body := map[string]any{"name": name}
	if description != "" {
		body["description"] = description
	}
	if err := f.apply(body, changed); err != nil {
		return nil, err
	}
	return body, nil
}

func runCreate(ctx context.Context, o *createOptions) error {
	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", "task/", o.body)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}

	var resp struct {
		ID int `json:"id"`
	}
	if err := cmdutil.DecodeJSON(raw, &resp); err != nil {
		return err
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", resp.ID)
		return nil
	}
	fmt.Fprintf(o.out, "Created task %d\n", resp.ID)
	return nil
}
