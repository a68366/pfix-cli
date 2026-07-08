package task

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/output"
)

func TestRunViewDefaultDetail(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		io.WriteString(w, `{"result":"success","task":{"id":15,"name":"Fix critical bug","description":"Something broke","status":{"name":"New"},"priority":"high","type":"bug","dateTime":{"datetime":"2024-01-15"},"overdue":false}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "15"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.HasPrefix(gotPath, "/task/15") {
		t.Errorf("path = %q, want /task/15...", gotPath)
	}
	if !strings.Contains(gotPath, "fields=") {
		t.Errorf("path missing fields param: %q", gotPath)
	}
	result := out.String()
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME: %q", result)
	}
	if !strings.Contains(result, "New") {
		t.Errorf("output missing status New: %q", result)
	}
	if !strings.Contains(result, "Fix critical bug") {
		t.Errorf("output missing task name: %q", result)
	}
}

func TestRunViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","task":{"id":15,"name":"Fix critical bug"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "15"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
}

func TestRunViewNonNumericID(t *testing.T) {
	// No HTTP request should be made; error must be returned immediately.
	requestSent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestSent = true
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		client: fakeClient(srv.URL),
		out:    out,
	}
	err := runView(context.Background(), o, "abc")
	if err == nil {
		t.Fatal("expected error on non-numeric id")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("error should mention 'number', got: %q", err.Error())
	}
	if requestSent {
		t.Error("no HTTP request should have been sent for invalid id")
	}
}

func TestRunViewCustomFields(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		io.WriteString(w, `{"result":"success","task":{"id":15,"name":"Task"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		fields: "id,name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "15"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	// url.QueryEscape encodes comma as %2C
	if !strings.Contains(gotPath, "id%2Cname") && !strings.Contains(gotPath, "id,name") {
		t.Errorf("path missing custom fields: %q", gotPath)
	}
	result := out.String()
	// custom fields use UPPER(field) headers
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID column: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME column: %q", result)
	}
}

func TestViewRendersCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","task":{"id":57,"name":"Task",`+
			`"customFieldData":[{"field":{"id":88206,"name":"test"},"value":"updated-value","stringValue":"updated-value"}]}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{fields: "id,name,88206", client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "57"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	got := out.String()
	if !strings.Contains(got, "test") || !strings.Contains(got, "updated-value") {
		t.Errorf("output missing custom-field row:\n%s", got)
	}
	// The numeric id must not appear as its own column header/label.
	for _, line := range strings.Split(got, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "88206") {
			t.Errorf("numeric id column not dropped:\n%s", got)
		}
	}
}

func TestViewNoCustomFields(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","task":{"id":57,"name":"Task"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "57"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.Contains(out.String(), "Task") {
		t.Errorf("standard fields missing:\n%s", out.String())
	}
}

// TestCustomFieldRowsKeyAbsent checks the very first guard in isolation: when
// the task map has no customFieldData key at all, customFieldRows must
// return a true nil slice (not just a zero-length one), matching the "no
// rows appended to Detail" contract runView relies on.
func TestCustomFieldRowsKeyAbsent(t *testing.T) {
	got := customFieldRows(map[string]any{"id": 1, "name": "Task"})
	if got != nil {
		t.Fatalf("customFieldRows() = %#v, want nil", got)
	}
}

// TestCustomFieldRows exercises customFieldRows directly against hand-built
// map[string]any inputs, covering each defensive branch: an empty
// customFieldData array, non-map array entries, entries with a missing/
// absent/non-string field.name, a well-formed entry with no stringValue, and
// a mixed array where only the valid entry survives.
func TestCustomFieldRows(t *testing.T) {
	tests := []struct {
		name string
		task map[string]any
		want []output.KV
	}{
		{
			name: "customFieldData is an empty array",
			task: map[string]any{"customFieldData": []any{}},
			want: nil,
		},
		{
			name: "array entries that are not maps are skipped",
			task: map[string]any{"customFieldData": []any{"not-a-map", 42, true}},
			want: nil,
		},
		{
			name: "entry missing field key is skipped",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"stringValue": "orphan"},
			}},
			want: nil,
		},
		{
			name: "entry whose field is not a map is skipped",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"field": "not-a-map", "stringValue": "x"},
			}},
			want: nil,
		},
		{
			name: "entry whose field.name is missing is skipped",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"field": map[string]any{"id": 88206}, "stringValue": "x"},
			}},
			want: nil,
		},
		{
			name: "entry whose field.name is non-string is skipped",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"field": map[string]any{"name": 123}, "stringValue": "x"},
			}},
			want: nil,
		},
		{
			name: "well-formed entry without stringValue renders an empty value",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"field": map[string]any{"id": 88206, "name": "Region"}},
			}},
			want: []output.KV{{Key: "Region", Value: ""}},
		},
		{
			name: "well-formed entry with stringValue renders name and value",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"field": map[string]any{"id": 88206, "name": "Region"}, "stringValue": "EU"},
			}},
			want: []output.KV{{Key: "Region", Value: "EU"}},
		},
		{
			name: "mixed array keeps only the valid entry",
			task: map[string]any{"customFieldData": []any{
				map[string]any{"field": map[string]any{"name": "Region"}, "stringValue": "EU"},
				"not-a-map",
				map[string]any{"field": map[string]any{"id": 2}},
			}},
			want: []output.KV{{Key: "Region", Value: "EU"}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := customFieldRows(tt.task)
			if len(got) != len(tt.want) {
				t.Fatalf("customFieldRows() = %#v, want %#v", got, tt.want)
			}
			for i, wantKV := range tt.want {
				if got[i] != wantKV {
					t.Errorf("row %d = %#v, want %#v", i, got[i], wantKV)
				}
			}
		})
	}
}
