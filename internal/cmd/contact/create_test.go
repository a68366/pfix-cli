package contact

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

func TestRunCreateAllFields(t *testing.T) {
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
		name:     "Pfix",
		lastname: "T",
		email:    "a@b.c",
		template: 1,
		client:   fakeClient(srv.URL),
		out:      out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/contact/" {
		t.Errorf("path = %q, want /contact/", gotPath)
	}
	if gotBody["name"] != "Pfix" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "Pfix")
	}
	if gotBody["lastname"] != "T" {
		t.Errorf("body lastname = %v, want %q", gotBody["lastname"], "T")
	}
	if gotBody["email"] != "a@b.c" {
		t.Errorf("body email = %v, want %q", gotBody["email"], "a@b.c")
	}
	tmpl, ok := gotBody["template"].(map[string]any)
	if !ok {
		t.Fatalf("body template = %v, want map", gotBody["template"])
	}
	if tmpl["id"] != float64(1) {
		t.Errorf("body template.id = %v, want 1", tmpl["id"])
	}
	if out.String() != "Created contact 42\n" {
		t.Errorf("output = %q, want %q", out.String(), "Created contact 42\n")
	}
}

func TestRunCreateOnlyRequired(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success","id":1}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		name:     "OnlyName",
		template: 5,
		client:   fakeClient(srv.URL),
		out:      out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if _, has := gotBody["lastname"]; has {
		t.Error("body should not include lastname when empty")
	}
	if _, has := gotBody["email"]; has {
		t.Error("body should not include email when empty")
	}
	if gotBody["name"] != "OnlyName" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "OnlyName")
	}
	tmpl, ok := gotBody["template"].(map[string]any)
	if !ok {
		t.Fatalf("body template = %v, want map", gotBody["template"])
	}
	if tmpl["id"] != float64(5) {
		t.Errorf("body template.id = %v, want 5", tmpl["id"])
	}
}

func TestRunCreateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","id":42}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		name:     "Pfix",
		template: 1,
		quiet:    true,
		client:   fakeClient(srv.URL),
		out:      out,
	}
	if err := runCreate(context.Background(), o); err != nil {
		t.Fatalf("runCreate: %v", err)
	}
	if out.String() != "42\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "42\n")
	}
}

func TestRunCreateJSON(t *testing.T) {
	raw := `{"result":"success","id":42}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, raw)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &createOptions{
		name:     "Pfix",
		template: 1,
		json:     true,
		client:   fakeClient(srv.URL),
		out:      out,
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

func TestCreateMissingName(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newCreateCmd(g)
	cmd.SetArgs([]string{"--template", "1"})
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

func TestCreateMissingTemplate(t *testing.T) {
	g := &cmdutil.GlobalOpts{}
	cmd := newCreateCmd(g)
	cmd.SetArgs([]string{"--name", "Pfix"})
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --template is missing")
	}
	if !strings.Contains(err.Error(), "template") {
		t.Errorf("error should mention the missing 'template' flag, got: %q", err.Error())
	}
}
