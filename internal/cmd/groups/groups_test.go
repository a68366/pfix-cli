package groups

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

func TestRunListUserGroups(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","groups":[{"id":23850,"name":"Everyone"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "user", client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/user/groups" {
		t.Errorf("path = %q, want /user/groups", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=id%2Cname") && !strings.Contains(gotQuery, "fields=id,name") {
		t.Errorf("query missing default fields: %q", gotQuery)
	}
	result := out.String()
	for _, want := range []string{"ID", "NAME", "23850", "Everyone"} {
		if !strings.Contains(result, want) {
			t.Errorf("output missing %q: %q", want, result)
		}
	}
}

func TestRunListContactGroupsPath(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		io.WriteString(w, `{"result":"success","groups":[]}`)
	}))
	defer srv.Close()

	o := &listOptions{objectType: "contact", client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if gotPath != "/contact/groups" {
		t.Errorf("path = %q, want /contact/groups", gotPath)
	}
}

func TestRunListJSONPassthrough(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","groups":[{"id":1,"name":"X"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "user", json: true, client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if !strings.Contains(out.String(), `"groups"`) {
		t.Errorf("json output missing envelope: %q", out.String())
	}
}

func TestRunListQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","groups":[{"id":1,"name":"X"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "user", quiet: true, client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	if strings.Contains(out.String(), "ID") || strings.Contains(out.String(), "NAME") {
		t.Errorf("quiet mode should not show headers: %q", out.String())
	}
	if !strings.Contains(out.String(), "X") {
		t.Errorf("quiet mode should still show data: %q", out.String())
	}
}
