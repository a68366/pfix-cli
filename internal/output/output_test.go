package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFlatten(t *testing.T) {
	row := map[string]any{
		"id":         float64(15), // json numbers decode to float64
		"name":       "Task A",
		"status":     map[string]any{"id": float64(1), "name": "New"},
		"thinStatus": map[string]any{"id": float64(2)},
		"strIDObj":   map[string]any{"id": "user:1"},
		"dateTime":   map[string]any{"datetime": "2026-06-29T21:46Z"},
		"assignees": []any{
			map[string]any{"name": "Ann"},
			map[string]any{"name": "Bob"},
		},
		"tags": []any{"x", "y"},
	}
	cases := map[string]string{
		"id":                "15",
		"name":              "Task A",
		"status.name":       "New",
		"status":            "New",    // object with name → name
		"thinStatus":        "2",      // object with only id → id fallback
		"strIDObj":          "user:1", // object with string id → id fallback
		"dateTime.datetime": "2026-06-29T21:46Z",
		"assignees":         "Ann, Bob", // array of objects → join names
		"tags":              "x, y",     // array of scalars → join
		"missing":           "",         // absent → empty
		"dateTime":          "",         // object without name → empty
	}
	for path, want := range cases {
		if got := Flatten(row, path); got != want {
			t.Errorf("Flatten(%q) = %q, want %q", path, got, want)
		}
	}
}

func TestTable(t *testing.T) {
	cols := []Column{{"ID", "id"}, {"NAME", "name"}, {"STATUS", "status.name"}}
	rows := []map[string]any{
		{"id": float64(1), "name": "A", "status": map[string]any{"name": "New"}},
		{"id": float64(2), "name": "B", "status": map[string]any{"name": "Done"}},
	}
	var b bytes.Buffer
	Table(&b, cols, rows, true)
	out := b.String()
	if !strings.Contains(out, "ID") || !strings.Contains(out, "NAME") {
		t.Fatalf("missing header: %q", out)
	}
	if !strings.Contains(out, "New") || !strings.Contains(out, "Done") {
		t.Fatalf("missing rows: %q", out)
	}
	// header suppressed
	b.Reset()
	Table(&b, cols, rows, false)
	if strings.Contains(b.String(), "ID\t") || strings.HasPrefix(b.String(), "ID") {
		t.Fatalf("header should be suppressed: %q", b.String())
	}
}

func TestJSONPretty(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, []byte(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(b.String(), "\n  \"a\": 1") {
		t.Fatalf("not pretty: %q", b.String())
	}
}

func TestJSONInvalidPassthrough(t *testing.T) {
	var b bytes.Buffer
	if err := JSON(&b, []byte("not json")); err != nil {
		t.Fatal(err)
	}
	if b.String() != "not json\n" {
		t.Fatalf("got %q", b.String())
	}
}

func TestDetail(t *testing.T) {
	cols := []Column{
		{"ID", "id"},
		{"NAME", "name"},
		{"STATUS", "status.name"},
		{"DESCRIPTION", "description"},
	}
	obj := map[string]any{
		"id":          float64(15),
		"name":        "A",
		"status":      map[string]any{"name": "New"},
		"description": "line1\nline2",
	}
	var b bytes.Buffer
	Detail(&b, cols, obj)
	out := b.String()

	// basic headers and flattened values
	for _, want := range []string{"ID", "15", "NAME", "A", "STATUS", "New"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q:\n%s", want, out)
		}
	}

	// newline in description must be collapsed: the literal sequence "line1\nline2"
	// (a real newline between them) must not appear; both words must be on one line
	if strings.Contains(out, "line1\nline2") {
		t.Errorf("newline inside cell was not cleaned:\n%s", out)
	}
	if !strings.Contains(out, "line1") || !strings.Contains(out, "line2") {
		t.Errorf("description words missing from output:\n%s", out)
	}
	// confirm they appear on the same output line (no newline between them)
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "line1") && strings.Contains(line, "line2") {
			break // found together on one line — good
		}
		if strings.Contains(line, "line1") {
			t.Errorf("line1 and line2 not on the same output line:\n%s", out)
			break
		}
	}
}

func TestColumnsFor(t *testing.T) {
	defCols := []Column{
		{Header: "ID", Path: "id"},
		{Header: "NAME", Path: "name"},
	}
	const def = "id,name"

	// When fields == def, should return defCols unchanged.
	got := ColumnsFor(def, def, defCols)
	if len(got) != len(defCols) || got[0] != defCols[0] || got[1] != defCols[1] {
		t.Errorf("ColumnsFor(def, def, defCols) = %v, want %v", got, defCols)
	}

	// Override CSV with spaces: headers should be trimmed and upper-cased.
	got = ColumnsFor("id, name ,status", def, defCols)
	want := []Column{
		{Header: "ID", Path: "id"},
		{Header: "NAME", Path: "name"},
		{Header: "STATUS", Path: "status"},
	}
	if len(got) != len(want) {
		t.Fatalf("len = %d, want %d; got %v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("col[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// Empty entries (e.g. trailing comma) should be skipped.
	got = ColumnsFor("id,,name", def, defCols)
	if len(got) != 2 {
		t.Errorf("expected 2 cols (empty entry skipped), got %d: %v", len(got), got)
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		name string
		in   string
		max  int
		want string
	}{
		{
			name: "truncated ascii",
			in:   "hello world",
			max:  5,
			want: "hello…", // 11 runes > 5 → cut + ellipsis
		},
		{
			name: "under limit multibyte",
			in:   "héllo",
			max:  100,
			want: "héllo", // 5 runes ≤ 100 → unchanged
		},
		{
			name: "max zero non-empty",
			in:   "hello",
			max:  0,
			want: "…", // 5 runes > 0 → r[:0] + ellipsis
		},
		{
			name: "exact boundary no ellipsis",
			in:   "hello",
			max:  5,
			want: "hello", // 5 runes == 5 → unchanged, no ellipsis
		},
		{
			name: "multibyte over limit rune boundary",
			in:   "héllo wörld", // 11 runes: h é l l o ' ' w ö r l d
			max:  5,
			want: "héllo…", // r[:5] = "héllo"
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Truncate(tt.in, tt.max)
			if got != tt.want {
				t.Errorf("Truncate(%q, %d) = %q, want %q", tt.in, tt.max, got, tt.want)
			}
		})
	}
}
