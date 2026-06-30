package contact

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
	client func() (*planfix.Client, error)
	out    io.Writer
}

func newUpdateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	var name, lastname, email, description string
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a contact",
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
			if cmd.Flags().Changed("lastname") {
				body["lastname"] = lastname
			}
			if cmd.Flags().Changed("email") {
				body["email"] = email
			}
			if cmd.Flags().Changed("description") {
				body["description"] = description
			}
			o := &updateOptions{
				id:     id,
				body:   body,
				json:   g.JSON,
				quiet:  g.Quiet,
				client: g.ClientFunc(),
				out:    cmd.OutOrStdout(),
			}
			return runUpdate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&name, "name", "", "Contact first name")
	cmd.Flags().StringVar(&lastname, "lastname", "", "Contact last name")
	cmd.Flags().StringVar(&email, "email", "", "Contact email address")
	cmd.Flags().StringVar(&description, "description", "", "Contact description")
	return cmd
}

func runUpdate(ctx context.Context, o *updateOptions) error {
	if len(o.body) == 0 {
		return fmt.Errorf("at least one of --name, --lastname, --email, or --description is required")
	}

	client, err := o.client()
	if err != nil {
		return err
	}
	path := "contact/" + strconv.Itoa(o.id)
	raw, err := client.JSON(ctx, "POST", path, o.body)
	if err != nil {
		return err
	}
	if o.json {
		return output.JSON(o.out, raw)
	}
	if o.quiet {
		fmt.Fprintf(o.out, "%d\n", o.id)
		return nil
	}
	fmt.Fprintf(o.out, "Updated contact %d\n", o.id)
	return nil
}
