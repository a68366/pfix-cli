package config

import (
	"fmt"
	"io"
	"os"
	"sort"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	pfconfig "github.com/a68366/pfix-cli/internal/config"
	"github.com/a68366/pfix-cli/internal/output"
)

var listColumns = []output.Column{
	{Header: "NAME", Path: "name"},
	{Header: "DOMAIN", Path: "domain"},
	{Header: "ACTIVE", Path: "active"},
}

func newListCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List configured profiles",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := configPath()
			if err != nil {
				return err
			}
			return runList(path, cmd.OutOrStdout(), g.Quiet)
		},
	}
}

func runList(path string, out io.Writer, quiet bool) error {
	cfg, err := pfconfig.Load(path)
	if err != nil {
		return err
	}
	if len(cfg.Profiles) == 0 {
		fmt.Fprintln(out, "No profiles configured. Run `pfix auth login` to add one.")
		return nil
	}
	active := pfconfig.ResolveProfileName("", os.Getenv, cfg)
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	rows := make([]map[string]any, 0, len(names))
	for _, name := range names {
		marker := ""
		if name == active {
			marker = "*"
		}
		rows = append(rows, map[string]any{
			"name":   name,
			"domain": cfg.Profiles[name].Domain,
			"active": marker,
		})
	}
	output.Table(out, listColumns, rows, !quiet)
	return nil
}
