package object

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
		io.WriteString(w, `{"result":"success","object":{"id":1,"name":"Obj1","description":"A test object","status":{"id":1,"name":"New"},"priority":"NotUrgent"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "1"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.HasPrefix(gotPath, "/object/1") {
		t.Errorf("path = %q, want /object/1...", gotPath)
	}
	if !strings.Contains(gotPath, "fields=") {
		t.Errorf("path missing fields param: %q", gotPath)
	}
	result := out.String()
	if !strings.Contains(result, "STATUS") {
		t.Errorf("output missing STATUS header: %q", result)
	}
	// Proves status.name path is used (not status.id)
	if !strings.Contains(result, "New") {
		t.Errorf("output missing status name New: %q", result)
	}
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME header: %q", result)
	}
	if !strings.Contains(result, "Obj1") {
		t.Errorf("output missing object name Obj1: %q", result)
	}
}

func TestRunViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","object":{"id":1,"name":"Obj1","status":{"id":1,"name":"New"},"priority":"NotUrgent"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "1"); err != nil {
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
		io.WriteString(w, `{"result":"success","object":{"id":1,"name":"Obj1"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		fields: "id,name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "1"); err != nil {
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
