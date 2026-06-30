package user

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunViewDefaultDetail(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		io.WriteString(w, `{"result":"success","user":{"id":4,"name":"Alice","midname":"","lastname":"Smith","email":"alice@example.com","login":"asmith","status":"active","role":"user","position":{"id":1,"title":"Engineer"}}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "4"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.HasPrefix(gotPath, "/user/4") {
		t.Errorf("path = %q, want /user/4...", gotPath)
	}
	if !strings.Contains(gotPath, "fields=") {
		t.Errorf("path missing fields param: %q", gotPath)
	}
	result := out.String()
	if !strings.Contains(result, "EMAIL") {
		t.Errorf("output missing EMAIL header: %q", result)
	}
	if !strings.Contains(result, "alice@example.com") {
		t.Errorf("output missing email value: %q", result)
	}
	if !strings.Contains(result, "LOGIN") {
		t.Errorf("output missing LOGIN header: %q", result)
	}
	if !strings.Contains(result, "asmith") {
		t.Errorf("output missing login value: %q", result)
	}
}

func TestRunViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","user":{"id":4,"name":"Alice","email":"alice@example.com","login":"asmith"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "4"); err != nil {
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
		io.WriteString(w, `{"result":"success","user":{"id":4,"name":"Alice"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		fields: "id,name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "4"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	// url.QueryEscape encodes comma as %2C
	if !strings.Contains(gotPath, "id%2Cname") && !strings.Contains(gotPath, "id,name") {
		t.Errorf("path missing custom fields: %q", gotPath)
	}
	result := out.String()
	if !strings.Contains(result, "ID") {
		t.Errorf("output missing ID column: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME column: %q", result)
	}
}
