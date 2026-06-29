package task

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
