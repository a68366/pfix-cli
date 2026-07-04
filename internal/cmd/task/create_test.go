package task

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func TestRunCreateDefault(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":42}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body:   map[string]any{"name": "New task", "description": "A description"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/" {
		t.Errorf("path = %q, want /task/", gotPath)
	}
	if gotBody["name"] != "New task" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "New task")
	}
	if gotBody["description"] != "A description" {
		t.Errorf("body description = %v, want %q", gotBody["description"], "A description")
	}
	result := out.String()
	if result != "Created task 42\n" {
		t.Errorf("output = %q, want %q", result, "Created task 42\n")
	}
}

func TestRunCreateSendsBodyVerbatim(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":1}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body: map[string]any{
			"name":     "Full",
			"template": map[string]any{"id": 6},
			"assignees": map[string]any{
				"users":  []any{map[string]any{"id": "user:1"}},
				"groups": []any{},
			},
			"endDateTime": map[string]any{"date": "20-07-2026", "time": "18:00"},
		},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	tmpl, ok := gotBody["template"].(map[string]any)
	if !ok || tmpl["id"] != float64(6) {
		t.Errorf("body template = %#v, want id 6", gotBody["template"])
	}
	asg, ok := gotBody["assignees"].(map[string]any)
	if !ok {
		t.Fatalf("body assignees = %#v, want map", gotBody["assignees"])
	}
	users, ok := asg["users"].([]any)
	if !ok || len(users) != 1 || users[0].(map[string]any)["id"] != "user:1" {
		t.Errorf("assignees users = %#v, want [{id user:1}]", asg["users"])
	}
	end, ok := gotBody["endDateTime"].(map[string]any)
	if !ok || end["date"] != "20-07-2026" || end["time"] != "18:00" {
		t.Errorf("endDateTime = %#v, want 20-07-2026 18:00", gotBody["endDateTime"])
	}
}

func TestRunCreateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":7}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body:   map[string]any{"name": "Task"},
		quiet:  true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if out.String() != "7\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "7\n")
	}
}

func TestRunCreateJSON(t *testing.T) {
	raw := `{"result":"success","id":99}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, raw)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		body:   map[string]any{"name": "Task"},
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
	if !strings.Contains(result, `"id"`) {
		t.Errorf("json output missing id field: %q", result)
	}
}

// TestCreateMissingName drives the Cobra command without --name and asserts the
// required-flag guard rejects it before RunE builds a client or sends a request.
func TestCreateMissingName(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newCreateCmd(g)
	cmd.SetArgs([]string{})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --name is missing")
	}
	if !strings.Contains(err.Error(), "name") {
		t.Errorf("error should mention the missing 'name' flag, got: %q", err.Error())
	}
}

// TestCreateInvalidFieldFailsFast drives the Cobra command with an invalid
// field value and asserts the parse error surfaces before any client is built
// or request sent (the helper's message — not a config/transport error —
// proves where execution stopped).
func TestCreateInvalidFieldFailsFast(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "bad priority", args: []string{"--name", "x", "--priority", "sky-high"}, wantErr: "invalid priority"},
		{name: "bad people ref", args: []string{"--name", "x", "--assignees", "12"}, wantErr: "invalid people reference"},
		{name: "bad date", args: []string{"--name", "x", "--end-date", "tomorrow"}, wantErr: "invalid date"},
		{name: "zero template", args: []string{"--name", "x", "--template", "0"}, wantErr: "--template must be a positive number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &cmdutil.GlobalOpts{}
			cmd := newCreateCmd(g)
			cmd.SetArgs(tt.args)
			cmd.SetOut(io.Discard)
			cmd.SetErr(io.Discard)
			err := cmd.Execute()
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want it to contain %q", err, tt.wantErr)
			}
		})
	}
}
