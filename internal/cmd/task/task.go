package task

import (
	"encoding/json"
	"strings"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/output"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// NewCmd builds the `task` command group.
func NewCmd(g *cmdutil.GlobalOpts) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "task",
		Short: "Work with Planfix tasks",
	}
	cmd.AddCommand(newListCmd(g), newViewCmd(g))
	return cmd
}

// clientFunc returns a configured client; overridable in tests via the options struct.
func clientFunc(g *cmdutil.GlobalOpts) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c, _, err := g.Client()
		return c, err
	}
}

// fieldsCSV returns override if non-empty, else def.
func fieldsCSV(override, def string) string {
	if override != "" {
		return override
	}
	return def
}

// columnsFor builds table columns from a comma-separated field list, using the
// provided default Column specs when fields == def (rich paths), or deriving
// header=UPPER(field), path=field for an explicit --fields override.
func columnsFor(fields, def string, defCols []output.Column) []output.Column {
	if fields == def {
		return defCols
	}
	parts := strings.Split(fields, ",")
	cols := make([]output.Column, 0, len(parts))
	for _, f := range parts {
		f = strings.TrimSpace(f)
		if f != "" {
			cols = append(cols, output.Column{Header: strings.ToUpper(f), Path: f})
		}
	}
	return cols
}

func jsonUnmarshal(b []byte, v any) error {
	return json.Unmarshal(b, v)
}
