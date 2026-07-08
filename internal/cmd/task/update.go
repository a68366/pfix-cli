package task

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
	id      int
	body    map[string]any
	cfSpecs []cmdutil.CustomFieldSpec
	json    bool
	quiet   bool
	jq      string
	client  func() (*planfix.Client, error)
	out     io.Writer
}

func newUpdateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	var name, description string
	f := &taskFields{}
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id, err := cmdutil.ValidateID(args[0])
			if err != nil {
				return err
			}
			body, err := updateBody(name, description, f, cmd.Flags().Changed)
			if err != nil {
				return err
			}
			specs, err := f.customFieldSpecs(cmd.Flags().Changed)
			if err != nil {
				return err
			}
			o := &updateOptions{
				id:      id,
				body:    body,
				cfSpecs: specs,
				json:    g.JSON,
				quiet:   g.Quiet,
				jq:      g.JQ,
				client:  g.ClientFunc(),
				out:     cmd.OutOrStdout(),
			}
			return runUpdate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Task name")
	cmd.Flags().StringVar(&description, "description", "", "Task description")
	f.register(cmd, false)
	return cmd
}

// updateBody assembles the update request body from every flag reported
// set by changed.
func updateBody(name, description string, f *taskFields, changed func(string) bool) (map[string]any, error) {
	body := map[string]any{}
	if changed("name") {
		body["name"] = name
	}
	if changed("description") {
		body["description"] = description
	}
	if err := f.apply(body, changed); err != nil {
		return nil, err
	}
	return body, nil
}

func runUpdate(ctx context.Context, o *updateOptions) error {
	if len(o.body) == 0 && len(o.cfSpecs) == 0 {
		return fmt.Errorf("at least one field flag is required (see --help)")
	}

	client, err := o.client()
	if err != nil {
		return err
	}
	if len(o.cfSpecs) > 0 {
		data, err := cmdutil.BuildCustomFieldData(ctx, client, "task", o.cfSpecs)
		if err != nil {
			return err
		}
		o.body["customFieldData"] = data
	}
	path := "task/" + strconv.Itoa(o.id)
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
	fmt.Fprintf(o.out, "Updated task %d\n", o.id)
	return nil
}
