package file

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

func TestViewPathAndDetail(t *testing.T) {
	var gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		io.WriteString(w, `{"result":"success","file":{"id":845086,"name":"doc.pdf","size":7}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "845086"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if gotPath != "/file/845086" {
		t.Errorf("path = %q", gotPath)
	}
	for _, want := range []string{"ID", "845086", "NAME", "doc.pdf", "SIZE", "7"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("detail missing %q: %q", want, out.String())
		}
	}
	// A file with no link must not render the LINK row; without this the
	// LINK column being always-on would go unnoticed.
	if strings.Contains(out.String(), "LINK") {
		t.Errorf("LINK row present for a linkless file: %q", out.String())
	}
}

func TestViewDetailWithLink(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","file":{"id":5,"name":"doc","size":0,"link":"https://x/doc"}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "5"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	for _, want := range []string{"LINK", "https://x/doc"} {
		if !strings.Contains(out.String(), want) {
			t.Errorf("detail missing %q: %q", want, out.String())
		}
	}
}

func TestViewJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","file":{"id":5,"name":"doc","size":0}}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &viewOptions{json: true, client: fakeClient(srv.URL), out: out}
	if err := runView(context.Background(), o, "5"); err != nil {
		t.Fatalf("runView: %v", err)
	}
	if !strings.Contains(out.String(), `"file"`) {
		t.Errorf("json output missing raw envelope: %q", out.String())
	}
}

func TestViewRejectsBadID(t *testing.T) {
	o := &viewOptions{client: fakeClient("http://unused"), out: &strings.Builder{}}
	if err := runView(context.Background(), o, "0"); err == nil {
		t.Fatal("want error for id 0")
	}
}
