package project

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
		name:        "New project",
		description: "A description",
		client:      fakeClient(srv.URL),
		out:         out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/project/" {
		t.Errorf("path = %q, want /project/", gotPath)
	}
	if gotBody["name"] != "New project" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "New project")
	}
	if gotBody["description"] != "A description" {
		t.Errorf("body description = %v, want %q", gotBody["description"], "A description")
	}
	result := out.String()
	if result != "Created project 42\n" {
		t.Errorf("output = %q, want %q", result, "Created project 42\n")
	}
}

func TestRunCreateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":7}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		name:   "Project",
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
		name:   "Project",
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

func TestRunCreateNoDescription(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":1}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		name:   "Only name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if _, hasDesc := gotBody["description"]; hasDesc {
		t.Error("body should not include description when empty")
	}
	if gotBody["name"] != "Only name" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "Only name")
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
