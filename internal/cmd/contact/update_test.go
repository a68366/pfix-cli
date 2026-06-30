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

func TestRunUpdateEmail(t *testing.T) {
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
		id:     4,
		body:   map[string]any{"email": "new@b.c"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/contact/4" {
		t.Errorf("path = %q, want /contact/4", gotPath)
	}
	if gotBody["email"] != "new@b.c" {
		t.Errorf("body email = %v, want %q", gotBody["email"], "new@b.c")
	}
	if _, has := gotBody["name"]; has {
		t.Error("body should not include name key when not set")
	}
	if _, has := gotBody["lastname"]; has {
		t.Error("body should not include lastname key when not set")
	}
	if _, has := gotBody["description"]; has {
		t.Error("body should not include description key when not set")
	}
	if out.String() != "Updated contact 4\n" {
		t.Errorf("output = %q, want %q", out.String(), "Updated contact 4\n")
	}
}

func TestRunUpdateNameAndLastname(t *testing.T) {
	var gotBody map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(b, &gotBody)
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     4,
		body:   map[string]any{"name": "N", "lastname": "L"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if gotBody["name"] != "N" {
		t.Errorf("body name = %v, want %q", gotBody["name"], "N")
	}
	if gotBody["lastname"] != "L" {
		t.Errorf("body lastname = %v, want %q", gotBody["lastname"], "L")
	}
}

func TestRunUpdateQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &updateOptions{
		id:     4,
		quiet:  true,
		body:   map[string]any{"email": "q@b.c"},
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runUpdate(context.Background(), o); err != nil {
		t.Fatalf("runUpdate: %v", err)
	}
	if out.String() != "4\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "4\n")
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
		id:     4,
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
		id:     4,
		body:   map[string]any{},
		client: fakeClient(srv.URL),
		out:    out,
	}
	err := runUpdate(context.Background(), o)
	if err == nil {
		t.Fatal("expected error when no field flags set")
	}
	const wantMsg = "at least one of --name, --lastname, --email, or --description is required"
	if err.Error() != wantMsg {
		t.Errorf("error = %q, want %q", err.Error(), wantMsg)
	}
	if requestSent {
		t.Error("no HTTP request should have been sent when body is empty")
	}
}

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
