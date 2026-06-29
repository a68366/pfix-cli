package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestFlatten(t *testing.T) {
	row := map[string]any{
		"id":       float64(15), // json numbers decode to float64
		"name":     "Task A",
		"status":   map[string]any{"id": float64(1), "name": "New"},
		"dateTime": map[string]any{"datetime": "2026-06-29T21:46Z"},
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
		"status":            "New", // object with name → name
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

func TestTruncate(t *testing.T) {
	if got := Truncate("hello world", 5); got != "hello…" {
		t.Fatalf("got %q", got)
	}
	if got := Truncate("héllo", 100); got != "héllo" {
		t.Fatalf("got %q", got)
	}
}
