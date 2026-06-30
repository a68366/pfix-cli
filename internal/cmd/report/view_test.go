package report

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunViewRepostEnvelope(t *testing.T) {
	// Planfix returns the report under the misspelled key "repost"; verify it is handled.
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.String()
		io.WriteString(w, `{"result":"success","repost":{"id":7,"name":"R7"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "7"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.HasPrefix(gotPath, "/report/7") {
		t.Errorf("path = %q, want /report/7...", gotPath)
	}
	if !strings.Contains(gotPath, "fields=") {
		t.Errorf("path missing fields param: %q", gotPath)
	}
	result := out.String()
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME header: %q", result)
	}
	if !strings.Contains(result, "R7") {
		t.Errorf("output missing name value R7: %q", result)
	}
}

func TestRunViewReportEnvelopeFallback(t *testing.T) {
	// Verify the fallback to the correctly-spelled "report" key also works.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","report":{"id":7,"name":"R7"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "7"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, "NAME") {
		t.Errorf("output missing NAME header: %q", result)
	}
	if !strings.Contains(result, "R7") {
		t.Errorf("output missing name value R7 (report fallback): %q", result)
	}
}

func TestRunViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","repost":{"id":7,"name":"R7"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		json:   true,
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "7"); err != nil {
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
		io.WriteString(w, `{"result":"success","repost":{"id":7,"name":"R7"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{
		fields: "id,name",
		client: fakeClient(srv.URL),
		out:    out,
	}
	if err := runView(context.Background(), o, "7"); err != nil {
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
