package cmd

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

func fakePingClient(url string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = url
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func TestRunPingSuccess(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &pingOptions{client: fakePingClient(srv.URL), out: out}
	if err := runPing(context.Background(), o); err != nil {
		t.Fatalf("runPing: %v", err)
	}
	if gotMethod != "GET" || gotPath != "/ping" {
		t.Errorf("request = %s %s, want GET /ping", gotMethod, gotPath)
	}
	if strings.TrimSpace(out.String()) != "OK" {
		t.Errorf("output = %q, want OK", out.String())
	}
}

func TestRunPingJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &pingOptions{json: true, client: fakePingClient(srv.URL), out: out}
	if err := runPing(context.Background(), o); err != nil {
		t.Fatalf("runPing: %v", err)
	}
	if !strings.Contains(out.String(), `"result"`) {
		t.Errorf("json output missing result field: %q", out.String())
	}
}

func TestRunPingQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &pingOptions{quiet: true, client: fakePingClient(srv.URL), out: out}
	if err := runPing(context.Background(), o); err != nil {
		t.Fatalf("runPing: %v", err)
	}
	if out.String() != "" {
		t.Errorf("quiet should print nothing, got %q", out.String())
	}
}

func TestRunPingUnknownTokenHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		io.WriteString(w, `{"result":"fail","code":1,"error":"Access denied, unknown token"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &pingOptions{client: fakePingClient(srv.URL), out: out}
	err := runPing(context.Background(), o)
	if err == nil {
		t.Fatal("expected an error on HTTP 401")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("error should carry the re-login hint: %v", err)
	}
}
