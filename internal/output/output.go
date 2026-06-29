// Package output renders Planfix API responses as human-readable tables and
// detail blocks, or as pretty-printed raw JSON.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
)

// Column maps a display header to a dot-path into a decoded JSON object.
type Column struct {
	Header string
	Path   string
}

// Flatten extracts the value at a dot-path from decoded JSON (map[string]any /
// []any / scalars) and renders it as a single-line string.
func Flatten(v any, path string) string {
	cur := v
	for _, key := range strings.Split(path, ".") {
		m, ok := cur.(map[string]any)
		if !ok {
			return ""
		}
		cur, ok = m[key]
		if !ok {
			return ""
		}
	}
	return render(cur)
}

func render(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return t
	case bool:
		return strconv.FormatBool(t)
	case float64:
		if t == float64(int64(t)) {
			return strconv.FormatInt(int64(t), 10)
		}
		return strconv.FormatFloat(t, 'f', -1, 64)
	case map[string]any:
		if name, ok := t["name"].(string); ok {
			return name
		}
		return ""
	case []any:
		parts := make([]string, 0, len(t))
		for _, e := range t {
			if s := render(e); s != "" {
				parts = append(parts, s)
			}
		}
		return strings.Join(parts, ", ")
	default:
		return fmt.Sprintf("%v", t)
	}
}

// Table writes rows as an aligned table. When showHeader is false the header
// row is omitted (for -q / scripting).
func Table(w io.Writer, cols []Column, rows []map[string]any, showHeader bool) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	if showHeader {
		hs := make([]string, len(cols))
		for i, c := range cols {
			hs[i] = c.Header
		}
		fmt.Fprintln(tw, strings.Join(hs, "\t"))
	}
	for _, row := range rows {
		cells := make([]string, len(cols))
		for i, c := range cols {
			cells[i] = clean(Flatten(row, c.Path))
		}
		fmt.Fprintln(tw, strings.Join(cells, "\t"))
	}
	tw.Flush()
}

// Detail writes a single object as aligned KEY value lines.
func Detail(w io.Writer, cols []Column, obj map[string]any) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	for _, c := range cols {
		fmt.Fprintf(tw, "%s\t%s\n", c.Header, clean(Flatten(obj, c.Path)))
	}
	tw.Flush()
}

// JSON pretty-prints raw JSON; non-JSON input is written verbatim (with a
// trailing newline).
func JSON(w io.Writer, raw []byte) error {
	if json.Valid(raw) {
		var buf bytes.Buffer
		if err := json.Indent(&buf, raw, "", "  "); err == nil {
			buf.WriteByte('\n')
			_, err := w.Write(buf.Bytes())
			return err
		}
	}
	if _, err := w.Write(raw); err != nil {
		return err
	}
	if len(raw) == 0 || raw[len(raw)-1] != '\n' {
		_, err := io.WriteString(w, "\n")
		return err
	}
	return nil
}

// ColumnsFor returns defCols when fields equals def (the rich, explicit-path
// defaults), otherwise derives one column per comma-separated field name
// (UPPER header, bare field as path).
func ColumnsFor(fields, def string, defCols []Column) []Column {
	if fields == def {
		return defCols
	}
	var cols []Column
	for _, f := range strings.Split(fields, ",") {
		f = strings.TrimSpace(f)
		if f == "" {
			continue
		}
		cols = append(cols, Column{Header: strings.ToUpper(f), Path: f})
	}
	return cols
}

// Truncate shortens s to max runes, appending an ellipsis when cut.
func Truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// clean collapses newlines/tabs so a cell stays on one table row.
func clean(s string) string {
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\t", " ")
	return strings.TrimSpace(s)
}
