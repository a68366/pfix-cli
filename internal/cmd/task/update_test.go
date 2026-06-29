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
	if !strings.Contains(err.Error(), "--name") || !strings.Contains(err.Error(), "--description") || !strings.Contains(err.Error(), "--status") {
		t.Errorf("error should mention field flags, got: %q", err.Error())
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
