package cmdutil

import (
	"fmt"
	"strings"
)

// fieldsHelpWrap is the soft column budget for the "Available fields" list;
// continuation lines are indented by fieldsHelpIndent.
const (
	fieldsHelpWrap   = 76
	fieldsHelpIndent = "  "
)

// FieldsHelp builds a command's Cobra Long help. It always renders short
// followed by a "Default fields" line. When available is non-empty it appends a
// word-wrapped "Available fields (N)" block, N being the number of
// comma-separated names in available. When cfResource is non-empty it appends a
// note that custom-field values are also selectable by numeric field id via
// `pfix customfield list <cfResource>`. Passing available == "" (default-only)
// or cfResource == "" (no note) is the intended way to omit each part.
func FieldsHelp(short, defaults, available, cfResource string) string {
	var b strings.Builder
	b.WriteString(short)
	b.WriteString("\n\n")
	b.WriteString("Default fields: ")
	b.WriteString(defaults)

	if available != "" {
		names := strings.Split(available, ",")
		fmt.Fprintf(&b, "\n\nAvailable fields (%d): ", len(names))
		writeWrapped(&b, names)
	}

	if cfResource != "" {
		b.WriteString("\n\nCustom-field values are also selectable by numeric field id\n")
		fmt.Fprintf(&b, "(see 'pfix customfield list %s'). ", cfResource)
		b.WriteString("Pass --fields to override the defaults.")
	}

	return b.String()
}

// writeWrapped writes names joined by ", ", breaking to a fresh
// fieldsHelpIndent-prefixed line before a name would overflow fieldsHelpWrap.
// The first line's already-written prefix ("Available fields (N): ") is treated
// as its starting column.
func writeWrapped(b *strings.Builder, names []string) {
	col := len("Available fields (): ") + len(fmt.Sprintf("%d", len(names)))
	for i, n := range names {
		token := n
		if i > 0 {
			token = ", " + n
		}
		if i > 0 && col+len(token) > fieldsHelpWrap {
			b.WriteString(",\n")
			b.WriteString(fieldsHelpIndent)
			b.WriteString(n)
			col = len(fieldsHelpIndent) + len(n)
			continue
		}
		b.WriteString(token)
		col += len(token)
	}
}
