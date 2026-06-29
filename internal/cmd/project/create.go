package project

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
		Short: "Create a project",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.quiet = g.Quiet
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runCreate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&o.name, "name", "", "Project name (required)")
	cmd.Flags().StringVar(&o.description, "description", "", "Project description")
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
	raw, err := client.JSON(ctx, "POST", "project/", body)
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
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
	fmt.Fprintf(o.out, "Created project %d\n", resp.ID)
	return nil
}
