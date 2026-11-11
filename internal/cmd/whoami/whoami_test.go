package whoami

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/planfix"
)

const userBody = `{"sub":"acme:user:1","account_name":"acme","name":"Jane Doe","groups":[{"name":"Developers","id":17},{"name":"Support","id":18}],"id":"user:1","type":"user","login":"jdoe","email":"jdoe@example.com","picture":"https://acme.example/avatar.png"}`

const contactBody = `{"sub":"acme:contact:42","account_name":"acme","name":null,"id":"contact:42","type":"contact","login":"client42","email":null,"picture":null}`

func fakeClient(url string) func() (*planfix.Client, error) {
	return func() (*planfix.Client, error) {
		c := planfix.New("example.test", "tok")
		c.BaseURL = url
		c.Limiter = rate.NewLimiter(rate.Inf, 1)
		c.Backoff = func(int) time.Duration { return 0 }
		return c, nil
	}
}

func serve(t *testing.T, status int, body string) (*httptest.Server, *string) {
	t.Helper()
	var got string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Method + " " + r.URL.Path
		w.WriteHeader(status)
		_, _ = io.WriteString(w, body)
	}))
	t.Cleanup(srv.Close)
	return srv, &got
}

func TestRunWhoamiDetail(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    []string
		notWant []string
	}{
		{
			name: "employee login",
			body: userBody,
			want: []string{"ID", "user:1", "NAME", "Jane Doe", "LOGIN", "jdoe", "EMAIL", "jdoe@example.com",
				"TYPE", "user", "ACCOUNT", "acme", "GROUPS", "Developers, Support"},
		},
		{
			name:    "contact login without groups and null fields",
			body:    contactBody,
			want:    []string{"contact:42", "client42", "contact", "acme"},
			notWant: []string{"NAME", "EMAIL", "GROUPS", "<nil>"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, got := serve(t, 200, tt.body)
			out := &strings.Builder{}
			if err := run(context.Background(), &options{client: fakeClient(srv.URL), out: out}); err != nil {
				t.Fatalf("run: %v", err)
			}
			if *got != "GET /userinfo" {
				t.Errorf("request = %q, want GET /userinfo", *got)
			}
			s := out.String()
			for _, w := range tt.want {
				if !strings.Contains(s, w) {
					t.Errorf("output missing %q:\n%s", w, s)
				}
			}
			for _, w := range tt.notWant {
				if strings.Contains(s, w) {
					t.Errorf("output should not contain %q:\n%s", w, s)
				}
			}
		})
	}
}

func TestRunWhoamiQuietPrintsID(t *testing.T) {
	srv, _ := serve(t, 200, userBody)
	out := &strings.Builder{}
	if err := run(context.Background(), &options{quiet: true, client: fakeClient(srv.URL), out: out}); err != nil {
		t.Fatalf("run: %v", err)
	}
	if out.String() != "user:1\n" {
		t.Errorf("quiet output = %q, want %q", out.String(), "user:1\n")
	}
}

func TestRunWhoamiJSON(t *testing.T) {
	tests := []struct {
		name string
		jq   string
		want string
	}{
		{name: "raw passthrough", want: `"account_name": "acme"`},
		{name: "jq filter", jq: ".groups | map(.id)", want: "17"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := serve(t, 200, userBody)
			out := &strings.Builder{}
			o := &options{json: true, jq: tt.jq, client: fakeClient(srv.URL), out: out}
			if err := run(context.Background(), o); err != nil {
				t.Fatalf("run: %v", err)
			}
			if !strings.Contains(out.String(), tt.want) {
				t.Errorf("output = %q, want it to contain %q", out.String(), tt.want)
			}
		})
	}
}

// TestRunWhoamiErrors runs each error through the root's description too
// (cmdutil.DescribeAPIError), since that is what the user sees: the /userinfo
// 403 hint and the shared auth hints must each appear once, never together.
func TestRunWhoamiErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   []string
		reject []string
	}{
		{name: "unsupported login type", status: 403, body: `{"result":"fail","code":0,"error":"Not supported"}`, want: []string{"Not supported — the API does not support this login type"}, reject: []string{"auth login", "scope"}},
		{name: "unknown token", status: 401, body: `{"result":"fail","code":1,"error":"Access denied, unknown token"}`, want: []string{"Access denied, unknown token — the token was rejected; run `pfix auth login --profile test`"}, reject: []string{"login type"}},
		{name: "scope denied on 403 gets no login-type hint", status: 403, body: `{"result":"fail","code":5,"error":"Scope denied"}`, want: []string{"Scope denied — the token is valid but lacks the scope"}, reject: []string{"login type"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv, _ := serve(t, tt.status, tt.body)
			err := run(context.Background(), &options{client: fakeClient(srv.URL), out: io.Discard})
			if err == nil {
				t.Fatal("expected an error")
			}
			msg := cmdutil.DescribeAPIError(err, cmdutil.AuthContext{Profile: "test"}).Error()
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("error = %q, want it to contain %q", msg, w)
				}
			}
			for _, r := range tt.reject {
				if strings.Contains(msg, r) {
					t.Errorf("error = %q, must not contain %q", msg, r)
				}
			}
			if n := strings.Count(msg, " — "); n != 1 {
				t.Errorf("error = %q carries %d hints, want 1", msg, n)
			}
			var apiErr *planfix.APIError
			if !errors.As(err, &apiErr) {
				t.Errorf("error should wrap *planfix.APIError: %v", err)
			}
		})
	}
}

func TestSummary(t *testing.T) {
	tests := []struct {
		name string
		info Info
		want string
	}{
		{name: "full", info: Info{ID: "user:1", Name: "Jane Doe", Login: "jdoe"}, want: "Jane Doe (user:1, login jdoe)"},
		{name: "no name", info: Info{ID: "contact:42", Login: "client42"}, want: "contact:42 (login client42)"},
		{name: "id only", info: Info{ID: "user:7"}, want: "user:7"},
		{name: "name without login", info: Info{ID: "user:7", Name: "Ann"}, want: "Ann (user:7)"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.info.Summary(); got != tt.want {
				t.Errorf("Summary() = %q, want %q", got, tt.want)
			}
		})
	}
}
