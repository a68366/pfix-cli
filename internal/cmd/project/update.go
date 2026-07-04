package project

import (
	"context"
	"fmt"
	"io"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

type updateOptions struct {
	id     int
	body   map[string]any
	json   bool
	quiet  bool
	jq     string
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newUpdateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	var name, description string
	var status int
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a project",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			body := map[string]any{}
			if cmd.Flags().Changed("name") {
				body["name"] = name
			}
			if cmd.Flags().Changed("description") {
				body["description"] = description
			}
			if cmd.Flags().Changed("status") {
				body["status"] = map[string]any{"id": status}
			}
			o := &updateOptions{
				id:     id,
				body:   body,
				json:   g.JSON,
				quiet:  g.Quiet,
				jq:     g.JQ,
				client: g.ClientFunc(),
				out:    cmd.OutOrStdout(),
			}
			return runUpdate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Project name")
	cmd.Flags().StringVar(&description, "description", "", "Project description")
	cmd.Flags().IntVar(&status, "status", 0, "Status ID")
	return cmd
}

func runUpdate(ctx context.Context, o *updateOptions) error {
	if len(o.body) == 0 {
		return fmt.Errorf("at least one of --name, --description, or --status is required")
	}

	client, err := o.client()
	if err != nil {
		return err
	}
	path := "project/" + strconv.Itoa(o.id)
	raw, err := client.JSON(ctx, "POST", path, o.body)
	if err != nil {
		return err
	}
	if o.json {
		return output.EmitJSON(o.out, raw, o.jq)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", o.id)
		return nil
	}
	fmt.Fprintf(o.out, "Updated project %d\n", o.id)
	return nil
}
