package planfix_test

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/planfix"
)

func fastClient(baseURL string) *planfix.Client {
	c := planfix.New("example.planfix.com", "secret")
	c.BaseURL = baseURL
	c.Limiter = rate.NewLimiter(rate.Inf, 1)
	c.Backoff = func(int) time.Duration { return 0 }
	return c
}

func TestDoSetsAuthHeaderAndPath(t *testing.T) {
	var gotAuth, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		io.WriteString(w, `{}`)
	}))
	defer srv.Close()

	resp, err := fastClient(srv.URL).Do(context.Background(), "GET", "task/123", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if gotAuth != "Bearer secret" {
		t.Errorf("auth header = %q", gotAuth)
	}
	if gotPath != "/task/123" {
		t.Errorf("path = %q", gotPath)
	}
}

func TestDoRetriesOn5xxThenSucceeds(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		io.WriteString(w, `{"ok":true}`)
	}))
	defer srv.Close()

	resp, err := fastClient(srv.URL).Do(context.Background(), "GET", "task/list", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 2 {
		t.Fatalf("calls = %d, want 2", got)
	}
}

func TestDoRequiresPositiveRetries(t *testing.T) {
	c := planfix.New("example.planfix.com", "secret")
	c.Retries = 0

	resp, err := c.Do(context.Background(), "GET", "task/1", nil, nil)
	if err == nil {
		t.Fatal("Do: want error, got nil")
	}
	if resp != nil {
		t.Errorf("resp = %v, want nil", resp)
	}
}

func TestURL(t *testing.T) {
	c := planfix.New("example.planfix.com", "secret")
	want := "https://example.planfix.com/rest/task/1"
	for _, in := range []string{"task/1", "/task/1", "/rest/task/1"} {
		if got := c.URL(in); got != want {
			t.Errorf("URL(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDoReturnsFinal5xxAfterExhaustion(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := fastClient(srv.URL)
	resp, err := c.Do(context.Background(), "GET", "task/1", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 500 {
		t.Errorf("status = %d, want 500", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != int32(c.Retries) {
		t.Errorf("calls = %d, want %d", got, c.Retries)
	}
}

func TestDoDoesNotRetryOn4xx(t *testing.T) {
	var calls int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		w.WriteHeader(http.StatusBadRequest)
		io.WriteString(w, `{"error":"bad"}`)
	}))
	defer srv.Close()

	resp, err := fastClient(srv.URL).Do(context.Background(), "GET", "x", nil, nil)
	if err != nil {
		t.Fatalf("Do: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != 400 {
		t.Errorf("status = %d, want 400", resp.StatusCode)
	}
	if got := atomic.LoadInt32(&calls); got != 1 {
		t.Errorf("calls = %d, want 1", got)
	}
}
