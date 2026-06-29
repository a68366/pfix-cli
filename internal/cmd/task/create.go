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
	name        string
	description string
	json        bool
	quiet       bool
	client      func() (*planfix.Client, error)
	out         io.Writer
}

func newCreateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a task",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.quiet = g.Quiet
			o.client = clientFunc(g)
			o.out = cmd.OutOrStdout()
			return runCreate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&o.name, "name", "", "Task name (required)")
	cmd.Flags().StringVar(&o.description, "description", "", "Task description")
	_ = cmd.MarkFlagRequired("name")
	return cmd
}

func runCreate(ctx context.Context, o *createOptions) error {
	body := map[string]any{
		"name": o.name,
	}
	if o.description != "" {
		body["description"] = o.description
	}

	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", "task/", body)
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
	fmt.Fprintf(o.out, "Created task %d\n", resp.ID)
	return nil
}
