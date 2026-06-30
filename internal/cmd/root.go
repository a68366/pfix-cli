package cmd

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/api"
	"github.com/a68366/pfix-cli/internal/cmd/auth"
	"github.com/a68366/pfix-cli/internal/cmd/config"
	"github.com/a68366/pfix-cli/internal/cmd/contact"
	"github.com/a68366/pfix-cli/internal/cmd/project"
	"github.com/a68366/pfix-cli/internal/cmd/report"
	"github.com/a68366/pfix-cli/internal/cmd/task"
	"github.com/a68366/pfix-cli/internal/cmd/user"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewRootCmd builds the root command tree.
func NewRootCmd() *cobra.Command {
	g := &cmdutil.GlobalOpts{}

	root := &cobra.Command{
		Use:           "pfix",
		Short:         "Command-line client for the Planfix REST API",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := root.PersistentFlags()
	pf.StringVar(&g.Profile, "profile", "", "Configuration profile to use")
	pf.StringVar(&g.Domain, "domain", "", "Planfix account domain (overrides the profile)")
	pf.BoolVar(&g.JSON, "json", false, "Emit raw JSON from the API instead of a table")
	pf.StringVar(&g.Fields, "fields", "", "Comma-separated fields to request (overrides defaults)")
	pf.BoolVarP(&g.Quiet, "quiet", "q", false, "Suppress non-essential output")

	root.AddCommand(newVersionCmd())
	root.AddCommand(auth.NewCmd(g))
	root.AddCommand(api.NewCmd(g))
	root.AddCommand(config.NewCmd(g))
	root.AddCommand(task.NewCmd(g))
	root.AddCommand(project.NewCmd(g))
	root.AddCommand(contact.NewCmd(g))
	root.AddCommand(user.NewCmd(g))
	root.AddCommand(report.NewCmd(g))

	return root
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
