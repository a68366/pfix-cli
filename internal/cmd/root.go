package cmd

import (
	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmd/api"
	"github.com/a68366/pfix-cli/internal/cmd/auth"
	"github.com/a68366/pfix-cli/internal/cmd/config"
	"github.com/a68366/pfix-cli/internal/cmd/contact"
	"github.com/a68366/pfix-cli/internal/cmd/customfield"
	"github.com/a68366/pfix-cli/internal/cmd/datatag"
	"github.com/a68366/pfix-cli/internal/cmd/object"
	"github.com/a68366/pfix-cli/internal/cmd/project"
	"github.com/a68366/pfix-cli/internal/cmd/report"
	"github.com/a68366/pfix-cli/internal/cmd/task"
	"github.com/a68366/pfix-cli/internal/cmd/template"
	"github.com/a68366/pfix-cli/internal/cmd/user"
	"github.com/a68366/pfix-cli/internal/cmdutil"
)

// NewRootCmd builds the root command tree.
func NewRootCmd() *cobra.Command {
	g := &cmdutil.GlobalOpts{}

	root := &cobra.Command{
		Use:   "pfix",
		Short: "Unofficial command-line client for the Planfix REST API",
		Long: "pfix is an unofficial command-line client for the Planfix REST API.\n" +
			"It is an independent open-source project, not affiliated with or endorsed by Planfix.",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pf := root.PersistentFlags()
	pf.StringVar(&g.Profile, "profile", "", "Configuration profile to use")
	pf.StringVar(&g.Domain, "domain", "", "Planfix account domain (overrides the profile)")
	pf.BoolVar(&g.JSON, "json", false, "Emit raw JSON from the API instead of a table")
	pf.StringVar(&g.Fields, "fields", "", "Comma-separated fields to request (overrides defaults)")
	pf.BoolVarP(&g.Quiet, "quiet", "q", false, "Suppress non-essential output")
	pf.StringVar(&g.JQ, "jq", "", "Filter JSON output with a jq expression")

	root.PersistentPreRunE = func(cmd *cobra.Command, args []string) error {
		return g.PreRun()
	}

	root.AddCommand(newVersionCmd())
	root.AddCommand(newPingCmd(g))
	root.AddCommand(auth.NewCmd(g))
	root.AddCommand(api.NewCmd(g))
	root.AddCommand(config.NewCmd(g))
	root.AddCommand(task.NewCmd(g))
	root.AddCommand(project.NewCmd(g))
	root.AddCommand(contact.NewCmd(g))
	root.AddCommand(user.NewCmd(g))
	root.AddCommand(datatag.NewCmd(g))
	root.AddCommand(report.NewCmd(g))
	root.AddCommand(template.NewCmd(g))
	root.AddCommand(customfield.NewCmd(g))
	root.AddCommand(object.NewCmd(g))

	return root
}

// Execute runs the root command.
func Execute() error {
	return NewRootCmd().Execute()
}
