package comment

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/planfix"
)

// fakeClient points a real client at an httptest server, with throttling and
// backoff disabled so tests do not sleep.
func fakeClient(srvURL string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = srvURL
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func TestViewRequestsFieldsAndRendersTaskComment(t *testing.T) {
	var gotPath, gotFields, gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotMethod = r.URL.Path, r.Method
		gotFields = r.URL.Query().Get("fields")
		io.WriteString(w, `{"result":"success","comment":{"id":11849892,
			"dateTime":{"datetime":"2026-07-27T18:02Z"},
			"owner":{"id":"user:1","name":"Ivan"},
			"task":{"id":12},"description":"hello","isPinned":false,"isHidden":false}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "11849892"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/comment/11849892" {
		t.Errorf("path = %q", gotPath)
	}
	if gotFields != viewDefaultFields {
		t.Errorf("fields = %q, want %q", gotFields, viewDefaultFields)
	}
	for _, want := range []string{"ID", "11849892", "CREATED", "AUTHOR", "Ivan", "TASK", "12", "TEXT", "hello"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("detail missing %q:\n%s", want, out.String())
		}
	}
	// A task comment must not render an empty CONTACT row.
	if strings.Contains(out.String(), "CONTACT") {
		t.Errorf("CONTACT row present for a task comment:\n%s", out.String())
	}
}

func TestViewRendersContactComment(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","comment":{"id":11849990,
			"contact":{"id":"contact:3","name":"Support"},"description":"hi"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "11849990"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.Contains(out.String(), "CONTACT") || !strings.Contains(out.String(), "contact:3") {
		t.Errorf("contact row missing:\n%s", out.String())
	}
	if strings.Contains(out.String(), "TASK") {
		t.Errorf("TASK row present for a contact comment:\n%s", out.String())
	}
}

func TestViewFieldsOverrideDrivesColumns(t *testing.T) {
	var gotFields string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotFields = r.URL.Query().Get("fields")
		io.WriteString(w, `{"result":"success","comment":{"id":5,"description":"x"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{fields: "id,description", client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "5"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if gotFields != "id,description" {
		t.Errorf("fields = %q", gotFields)
	}
	if !strings.Contains(out.String(), "DESCRIPTION") {
		t.Errorf("override columns missing:\n%s", out.String())
	}
}

func TestViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","comment":{"id":5}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{json: true, client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "5"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.Contains(out.String(), `"comment"`) {
		t.Errorf("json output missing raw envelope: %q", out.String())
	}
}

// TestViewUnknownIDReturnsAPIMessage pins the cmdutil.DescribeAPIError wiring:
// an unrecognized comment id (app code 5000) must surface the API's own
// message, not a generic error.
func TestViewUnknownIDReturnsAPIMessage(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"result":"fail","code":5000,"error":"Comment not found by id - 999"}`)
	}))
	defer srv.Close()

	o := &viewOptions{client: fakeClient(srv.URL), out: &strings.Builder{}}
	err := runView(context.Background(), o, "999")
	if err == nil {
		t.Fatal("want error")
	}
	if !strings.Contains(err.Error(), "Comment not found by id - 999") {
		t.Errorf("API message lost: %v", err)
	}
}

func TestViewRejectsBadID(t *testing.T) {
	o := &viewOptions{client: fakeClient("http://unused"), out: &strings.Builder{}}
	if err := runView(context.Background(), o, "0"); err == nil {
		t.Fatal("want error for id 0")
	}
}
