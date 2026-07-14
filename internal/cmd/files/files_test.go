package files

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

func fakeClient(srvURL string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = srvURL
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func baseOpts(srvURL string, opts Options, out, errOut io.Writer) *listOptions {
	return &listOptions{
		Options: opts,
		id:      42,
		source:  "attached",
		client:  fakeClient(srvURL),
		out:     out,
		errOut:  errOut,
	}
}

func TestAttachedTaskPathAndColumns(t *testing.T) {
	var gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath, gotQuery = r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","files":[{"id":7,"name":"a.pdf","size":11}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task", DescriptionOnly: true}, out, &strings.Builder{})
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if gotPath != "/task/42/files" {
		t.Errorf("path = %q", gotPath)
	}
	if gotQuery != "" {
		t.Errorf("query = %q, want empty (no onlyFromDescription by default)", gotQuery)
	}
	for _, want := range []string{"ID", "NAME", "SIZE", "7", "a.pdf", "11"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("output missing %q: %q", want, out.String())
		}
	}
}

func TestAttachedDescriptionOnlyQuery(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","files":[]}`)
	}))
	defer srv.Close()

	o := baseOpts(srv.URL, Options{Type: "contact", DescriptionOnly: true}, &strings.Builder{}, &strings.Builder{})
	o.descriptionOnly = true
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(gotQuery, "onlyFromDescription=true") {
		t.Errorf("query = %q, want onlyFromDescription=true", gotQuery)
	}
}

func TestAttachedProjectPaging(t *testing.T) {
	var gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.RawQuery
		io.WriteString(w, `{"result":"success","files":[]}`)
	}))
	defer srv.Close()

	o := baseOpts(srv.URL, Options{Type: "project", Paging: true}, &strings.Builder{}, &strings.Builder{})
	o.limit, o.offset = 50, 100
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(gotQuery, "pageSize=50") || !strings.Contains(gotQuery, "offset=100") {
		t.Errorf("query = %q, want pageSize=50 & offset=100", gotQuery)
	}
}

func TestAttachedJSONRawEcho(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","files":[{"id":7}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task"}, out, &strings.Builder{})
	o.json = true
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(out.String(), `"files"`) {
		t.Errorf("json output missing envelope: %q", out.String())
	}
}

func TestAttachedEmptyNote(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","files":[]}`)
	}))
	defer srv.Close()

	errOut := &strings.Builder{}
	o := baseOpts(srv.URL, Options{Type: "task"}, &strings.Builder{}, errOut)
	if err := runFiles(context.Background(), o); err != nil {
		t.Fatalf("runFiles: %v", err)
	}
	if !strings.Contains(errOut.String(), "no files") {
		t.Errorf("stderr note missing: %q", errOut.String())
	}
}

func TestSourceValidation(t *testing.T) {
	o := baseOpts("http://unused", Options{Type: "task", DescriptionOnly: true}, &strings.Builder{}, &strings.Builder{})
	o.source = "bogus"
	if err := runFiles(context.Background(), o); err == nil || !strings.Contains(err.Error(), "invalid --source") {
		t.Fatalf("want invalid --source error, got %v", err)
	}

	o2 := baseOpts("http://unused", Options{Type: "task", DescriptionOnly: true}, &strings.Builder{}, &strings.Builder{})
	o2.source, o2.descriptionOnly = "inline", true
	if err := runFiles(context.Background(), o2); err == nil || !strings.Contains(err.Error(), "description-only") {
		t.Fatalf("want description-only+inline conflict, got %v", err)
	}

	o3 := baseOpts("http://unused", Options{Type: "project", Paging: true}, &strings.Builder{}, &strings.Builder{})
	o3.source, o3.pagingSet = "inline", true
	if err := runFiles(context.Background(), o3); err == nil || !strings.Contains(err.Error(), "limit") {
		t.Fatalf("want limit+inline conflict, got %v", err)
	}
}
