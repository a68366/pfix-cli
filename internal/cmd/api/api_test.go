package api

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

func optsFor(srvURL string, in io.Reader, out io.Writer) *apiOptions {
	return &apiOptions{
		method: "GET",
		in:     in,
		out:    out,
		client: func() (*planfix.Client, error) {
			c := planfix.New("example.planfix.com", "tok")
			c.BaseURL = srvURL
			c.Limiter = rate.NewLimiter(rate.Inf, 1)
			c.Backoff = func(int) time.Duration { return 0 }
			return c, nil
		},
	}
}

func TestRunAPIDefaultsToGet(t *testing.T) {
	var gotMethod string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		io.WriteString(w, `{"id":1}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	if err := runAPI(context.Background(), optsFor(srv.URL, nil, out), "task/1"); err != nil {
		t.Fatalf("runAPI: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if !strings.Contains(out.String(), `"id": 1`) {
		t.Errorf("output not pretty-printed: %q", out.String())
	}
}

func TestRunAPIPostsFieldsAsJSON(t *testing.T) {
	var gotMethod, gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	o := optsFor(srv.URL, nil, &strings.Builder{})
	o.fields = []string{"n=5", "flag=true"}
	if err := runAPI(context.Background(), o, "task/"); err != nil {
		t.Fatalf("runAPI: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if !strings.Contains(gotBody, `"n":5`) || !strings.Contains(gotBody, `"flag":true`) {
		t.Errorf("body = %q, want typed values", gotBody)
	}
}

func TestRunAPIReadsInputFromStdin(t *testing.T) {
	var gotBody string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		gotBody = string(b)
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	o := optsFor(srv.URL, strings.NewReader(`{"a":1}`), &strings.Builder{})
	o.inputFile = "-"
	if err := runAPI(context.Background(), o, "task/list"); err != nil {
		t.Fatalf("runAPI: %v", err)
	}
	if strings.TrimSpace(gotBody) != `{"a":1}` {
		t.Errorf("body = %q", gotBody)
	}
}

func TestRunAPIReturnsErrorOnNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":"nope"}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	err := runAPI(context.Background(), optsFor(srv.URL, nil, out), "task/1")
	if err == nil {
		t.Fatal("expected error on 400")
	}
	if !strings.Contains(out.String(), "nope") {
		t.Errorf("body should still be printed, got %q", out.String())
	}
}

func TestRunAPIRejectsInputWithFields(t *testing.T) {
	// --input combined with -F or -f must be rejected before any request is sent.
	requestSent := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestSent = true
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	o := optsFor(srv.URL, strings.NewReader(`{"a":1}`), &strings.Builder{})
	o.inputFile = "-"
	o.fields = []string{"n=5"}
	err := runAPI(context.Background(), o, "task/list")
	if err == nil {
		t.Fatal("expected error when --input is combined with -F")
	}
	if requestSent {
		t.Error("no HTTP request should have been sent")
	}

	// Also check with --raw-field.
	o2 := optsFor(srv.URL, strings.NewReader(`{}`), &strings.Builder{})
	o2.inputFile = "-"
	o2.rawFields = []string{"key=val"}
	requestSent = false
	if err2 := runAPI(context.Background(), o2, "task/list"); err2 == nil {
		t.Fatal("expected error when --input is combined with -f")
	}
	if requestSent {
		t.Error("no HTTP request should have been sent")
	}
}

func TestSplitAndMagicValue(t *testing.T) {
	k, v, err := splitField("name=hello")
	if err != nil || k != "name" || v != "hello" {
		t.Fatalf("splitField = (%q,%q,%v)", k, v, err)
	}
	if got, _ := magicValue("42", nil); got != 42 {
		t.Errorf("magicValue int = %v", got)
	}
	if got, _ := magicValue("true", nil); got != true {
		t.Errorf("magicValue bool = %v", got)
	}
}
