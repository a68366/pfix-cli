package auth

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

func fakeStatusClient(url string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = url
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func TestRunStatusValidProbesPing(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		io.WriteString(w, `{"result":"success"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &statusOptions{
		profileName: "default",
		domain:      "acme.planfix.ru",
		token:       "supersecrettoken",
		client:      fakeStatusClient(srv.URL),
		out:         out,
	}
	if err := runStatus(context.Background(), o); err != nil {
		t.Fatalf("runStatus: %v", err)
	}
	if gotMethod != "GET" || gotPath != "/ping" {
		t.Errorf("validity probe = %s %s, want GET /ping", gotMethod, gotPath)
	}
	s := out.String()
	for _, want := range []string{"Profile: default", "Domain:  acme.planfix.ru", "Status:  valid"} {
		if !strings.Contains(s, want) {
			t.Errorf("output missing %q: %q", want, s)
		}
	}
	if strings.Contains(s, "supersecrettoken") {
		t.Errorf("token must be masked, but the raw token appeared: %q", s)
	}
}

func TestRunStatusUnknownTokenHint(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		io.WriteString(w, `{"result":"fail","code":1,"error":"Access denied, unknown token"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &statusOptions{
		profileName: "default",
		domain:      "acme.planfix.ru",
		token:       "tok",
		client:      fakeStatusClient(srv.URL),
		out:         out,
	}
	err := runStatus(context.Background(), o)
	if err == nil {
		t.Fatal("expected an error on HTTP 401")
	}
	if !strings.Contains(err.Error(), "auth login") {
		t.Errorf("error should carry the re-login hint: %v", err)
	}
}
