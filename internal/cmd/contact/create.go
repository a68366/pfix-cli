package contact

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
	name, lastname, email string
	template              int
	json                  bool
	quiet                 bool
	jq                    string
	client                func() (*planfix.Client, error)
	out                   io.Writer
}

func newCreateCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	o := &createOptions{}
	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a contact",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			o.json = g.JSON
			o.quiet = g.Quiet
			o.jq = g.JQ
			o.client = g.ClientFunc()
			o.out = cmd.OutOrStdout()
			return runCreate(cmd.Context(), o)
		},
	}
	cmd.Flags().StringVar(&o.name, "name", "", "Contact first name (required)")
	cmd.Flags().IntVar(&o.template, "template", 0, "Contact template ID (required; find via 'pfix contact list --fields template')")
	cmd.Flags().StringVar(&o.lastname, "lastname", "", "Contact last name")
	cmd.Flags().StringVar(&o.email, "email", "", "Contact email address")
	_ = cmd.MarkFlagRequired("name")
	_ = cmd.MarkFlagRequired("template")
	return cmd
}

func runCreate(ctx context.Context, o *createOptions) error {
	body := map[string]any{
		"template": map[string]any{"id": o.template},
		"name":     o.name,
	}
	if o.lastname != "" {
		body["lastname"] = o.lastname
	}
	if o.email != "" {
		body["email"] = o.email
	}

	client, err := o.client()
	if err != nil {
		return err
	}
	raw, err := client.JSON(ctx, "POST", "contact/", body)
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
	fmt.Fprintf(o.out, "Created contact %d\n", resp.ID)
	return nil
}
