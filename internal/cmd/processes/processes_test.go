package processes

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

// fakeClient returns a client factory pointed at srvURL with no rate limiting
// and zero backoff (hermetic, fast).
func fakeClient(srvURL string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = srvURL
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func TestRunListTaskProcesses(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		io.WriteString(w, `{"result":"success","processes":[{"id":234012,"name":"Alpha"},{"id":234106,"name":"Beta"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "task", client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/process/task" {
		t.Errorf("path = %q, want /process/task", gotPath)
	}
	result := out.String()
	if !strings.Contains(result, "Alpha") || !strings.Contains(result, "234106") {
		t.Errorf("output missing rows: %q", result)
	}
	for _, hdr := range []string{"ID", "NAME"} {
		if !strings.Contains(result, hdr) {
			t.Errorf("output missing header %q: %q", hdr, result)
		}
	}
}

func TestRunListContactPathAndFields(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","processes":[]}`)
	}))
	defer srv.Close()

	o := &listOptions{objectType: "contact", client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotPath != "/process/contact" {
		t.Errorf("path = %q, want /process/contact", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=id%2Cname") && !strings.Contains(gotQuery, "fields=id,name") {
		t.Errorf("query missing default fields: %q", gotQuery)
	}
}

func TestRunListJSONPassthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","processes":[{"id":1,"name":"X"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "task", json: true, client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(out.String(), `"processes"`) {
		t.Errorf("json output missing envelope: %q", out.String())
	}
}
