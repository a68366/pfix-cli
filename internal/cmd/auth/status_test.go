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
		if gotPath == "" {
			// The first request is the validity probe; /userinfo follows it.
			gotMethod, gotPath = r.Method, r.URL.Path
		}
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

func TestRunStatusUserLine(t *testing.T) {
	tests := []struct {
		name         string
		userStatus   int
		userBody     string
		wantUserLine string
	}{
		{
			name:         "userinfo success",
			userStatus:   200,
			userBody:     `{"sub":"acme:user:1","account_name":"acme","name":"Jane Doe","id":"user:1","type":"user","login":"jdoe"}`,
			wantUserLine: "User:    Jane Doe (user:1, login jdoe)",
		},
		{
			name:         "userinfo unsupported login type",
			userStatus:   403,
			userBody:     `{"result":"fail","code":0,"error":"Not supported"}`,
			wantUserLine: "User:    unavailable",
		},
		{
			name:         "userinfo server error",
			userStatus:   500,
			userBody:     `oops`,
			wantUserLine: "User:    unavailable",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/userinfo" {
					w.WriteHeader(tt.userStatus)
					io.WriteString(w, tt.userBody)
					return
				}
				io.WriteString(w, `{"result":"success"}`)
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
			if err := runStatus(context.Background(), o); err != nil {
				t.Fatalf("runStatus must not fail on a /userinfo outcome: %v", err)
			}
			s := out.String()
			if !strings.Contains(s, "Status:  valid") {
				t.Errorf("output missing token status: %q", s)
			}
			if !strings.Contains(s, tt.wantUserLine) {
				t.Errorf("output missing %q: %q", tt.wantUserLine, s)
			}
		})
	}
}

func TestRunStatusSkipsUserinfoWhenTokenInvalid(t *testing.T) {
	var userinfoCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/userinfo" {
			userinfoCalled = true
		}
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
	if err := runStatus(context.Background(), o); err == nil {
		t.Fatal("expected an error on HTTP 401")
	}
	if userinfoCalled {
		t.Error("/userinfo must not be called when the token check failed")
	}
	if strings.Contains(out.String(), "User:") {
		t.Errorf("no User line expected after a failed token check: %q", out.String())
	}
}
