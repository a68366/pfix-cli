package comment

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDeleteRequiresForce(t *testing.T) {
	hit := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hit = true
	}))
	defer srv.Close()

	o := &deleteOptions{client: fakeClient(srv.URL), out: &strings.Builder{}}
	err := runDelete(context.Background(), o, "11849892")
	if err == nil {
		t.Fatal("want error without --force")
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("error should name --force: %v", err)
	}
	if hit {
		t.Error("no request may be sent without --force")
	}
}

func TestDeleteSendsDeleteAndReports(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &deleteOptions{force: true, client: fakeClient(srv.URL), out: out}
	if err := runDelete(context.Background(), o, "11849892"); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if gotMethod != "DELETE" {
		t.Errorf("method = %q, want DELETE", gotMethod)
	}
	if gotPath != "/comment/11849892" {
		t.Errorf("path = %q", gotPath)
	}
	if !strings.Contains(out.String(), "Deleted comment 11849892") {
		t.Errorf("output = %q", out.String())
	}
}

func TestDeleteQuietPrintsIDOnly(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &deleteOptions{force: true, quiet: true, client: fakeClient(srv.URL), out: out}
	if err := runDelete(context.Background(), o, "42"); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if out.String() != "42\n" {
		t.Errorf("quiet output = %q, want \"42\\n\"", out.String())
	}
}

// The API rejects deleting a task's first (description) comment with an opaque
// app code 0, so pfix names the cause.
func TestDeleteDescriptionCommentHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"result":"fail","code":0,"error":"Rest API error"}`)
	}))
	defer srv.Close()

	o := &deleteOptions{force: true, client: fakeClient(srv.URL), out: &strings.Builder{}}
	err := runDelete(context.Background(), o, "11849932")
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "description") {
		t.Errorf("hint missing from error: %v", err)
	}
}

// An unknown id (app code 5000) must pass through unchanged — the description
// hint would be wrong there.
func TestDeleteUnknownIDKeepsAPIMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"result":"fail","code":5000,"error":"Comment not found by id - 999"}`)
	}))
	defer srv.Close()

	o := &deleteOptions{force: true, client: fakeClient(srv.URL), out: &strings.Builder{}}
	err := runDelete(context.Background(), o, "999")
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "Comment not found") {
		t.Errorf("API message lost: %v", err)
	}
	if strings.Contains(err.Error(), "description") {
		t.Errorf("description hint wrongly applied: %v", err)
	}
}

func TestDeleteJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &deleteOptions{force: true, json: true, client: fakeClient(srv.URL), out: out}
	if err := runDelete(context.Background(), o, "42"); err != nil {
		t.Fatalf("runDelete: %v", err)
	}
	if !strings.Contains(out.String(), `"result"`) {
		t.Errorf("json passthrough missing: %q", out.String())
	}
}
