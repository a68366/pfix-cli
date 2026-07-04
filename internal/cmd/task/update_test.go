package task

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func TestRunUpdateStatus(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     15,
		body:   map[string]any{"status": map[string]any{"id": 2}},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/15" {
		t.Errorf("path = %q, want /task/15", gotPath)
	}
	status, ok := gotBody["status"].(map[string]any)
	if !ok {
		t.Fatalf("body status = %v, want map", gotBody["status"])
	}
	if status["id"] != float64(2) {
		t.Errorf("body status.id = %v, want 2", status["id"])
	}
	if out.String() != "Updated task 15\n" {
		t.Errorf("output = %q, want %q", out.String(), "Updated task 15\n")
	}
}

func TestRunUpdateNameAndDescription(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     20,
		body:   map[string]any{"name": "Updated", "description": "Details"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if gotBody["name"] != "Updated" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "Updated")
	}
	if gotBody["description"] != "Details" {
		t.Errorf("body description = %v, want %q", gotBody["description"], "Details")
	}
}

func TestRunUpdateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     15,
		quiet:  true,
		body:   map[string]any{"name": "Q"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if out.String() != "15\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "15\n")
	}
}

func TestRunUpdateJSON(t *testing.T) {
	raw := `{"result":"success"}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, raw)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     15,
		json:   true,
		body:   map[string]any{"name": "J"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) {
		t.Errorf("json output missing result field: %q", result)
	}
	if !strings.Contains(result, `"success"`) {
		t.Errorf("json output missing success value: %q", result)
	}
}

func TestRunUpdateNoFields(t *testing.T) {
	requestSent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestSent = true
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     15,
		body:   map[string]any{},
		client: fakeClient(srv.URL),
		out:    out,
	}
	err := runUpdate(context.Background(), o)
	if err == nil {
		t.Fatal("expected error when no field flags set")
	}
	if !strings.Contains(err.Error(), "at least one field flag is required") {
		t.Errorf("error should say a field flag is required, got: %q", err.Error())
	}
	if requestSent {
		t.Error("no HTTP request should have been sent when body is empty")
	}
}

// TestUpdateNonNumericID drives the Cobra command with a non-numeric id and a
// field flag, asserting RunE rejects the id (with the validateID message) before
// building a client or sending a request. The validateID error message — not a
// config/transport error — proves execution stopped at the id check.
func TestUpdateNonNumericID(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newUpdateCmd(g)
	cmd.SetArgs([]string{"abc", "--name", "x"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error on non-numeric id")
	}
	if !strings.Contains(err.Error(), "number") {
		t.Errorf("error should mention 'number', got: %q", err.Error())
	}
}

// TestUpdateHasNoTemplateFlag pins the create-only nature of --template: a
// task's template cannot be changed after creation.
func TestUpdateHasNoTemplateFlag(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newUpdateCmd(g)
	if cmd.Flags().Lookup("template") != nil {
		t.Error("update must not register --template")
	}
}

// TestUpdateInvalidFieldFailsFast drives the Cobra command with an invalid
// field value and asserts the parse error surfaces before any client is built
// or request sent.
func TestUpdateInvalidFieldFailsFast(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		wantErr string
	}{
		{name: "bad people ref", args: []string{"15", "--assignees", "team:3"}, wantErr: "invalid people reference"},
		{name: "bad date", args: []string{"15", "--start-date", "10-07-2026"}, wantErr: "invalid date"},
		{name: "bad counterparty", args: []string{"15", "--counterparty", "user:1"}, wantErr: "invalid counterparty"},
		{name: "zero project", args: []string{"15", "--project", "0"}, wantErr: "--project must be a positive number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := &cmdutil.GlobalOpts{}
			cmd := newUpdateCmd(g)
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

func TestUpdateBodyFromCommandLine(t *testing.T) {
	var name, description string
	f := &taskFields{}
	cmd := &cobra.Command{}
	cmd.Flags().StringVar(&name, "name", "", "")
	cmd.Flags().StringVar(&description, "description", "", "")
	f.register(cmd, false)
	if err := cmd.Flags().Parse([]string{
		"--name", "Renamed",
		"--assignees", "user:1,group:3",
		"--start-date", "2026-07-08 10:00",
		"--status", "2",
	}); err != nil {
		t.Fatalf("parse: %v", err)
	}
	body, err := updateBody(name, description, f, cmd.Flags().Changed)
	if err != nil {
		t.Fatalf("updateBody: %v", err)
	}
	want := map[string]any{
		"name": "Renamed",
		"assignees": map[string]any{
			"users":  []any{map[string]any{"id": "user:1"}},
			"groups": []any{map[string]any{"id": 3}},
		},
		"startDateTime": map[string]any{"date": "08-07-2026", "time": "10:00"},
		"status":        map[string]any{"id": 2},
	}
	if !reflect.DeepEqual(body, want) {
		t.Errorf("body = %#v, want %#v", body, want)
	}
}
