package cmd

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/time/rate"

	"github.com/a68366/pfix-cli/internal/cmdutil"
	"github.com/a68366/pfix-cli/internal/planfix"
)

// newAPIProbeRoot returns the real root tree plus an "apiprobe" subcommand
// that calls the API through a real client and returns its error untouched —
// the shape of every command that does not describe its own errors (task list,
// project view, …). Real commands build their client from the config file and
// always dial https://<domain>, which an httptest server cannot stand in for,
// hence the probe.
func newAPIProbeRoot(url string) (*cobra.Command, *cmdutil.GlobalOpts) {
	root, g := newRootCmd()
	root.AddCommand(&cobra.Command{
		Use:  "apiprobe",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			c := planfix.New("example.test", "tok")
			c.BaseURL = url
			c.Limiter = rate.NewLimiter(rate.Inf, 1)
			c.Backoff = func(int) time.Duration { return 0 }
			_, err := c.JSON(cmd.Context(), "POST", "task/list", map[string]any{"pageSize": 1})
			return err
		},
	})
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	return root, g
}

func TestRootAddsAuthHint(t *testing.T) {
	const loginHint = "the token was rejected; run `pfix auth login --profile test` to re-authenticate"
	tests := []struct {
		name    string
		status  int
		body    string
		env     string // PFIX_TOKEN
		want    []string
		notWant []string
	}{
		{
			name:   "401 with an empty body",
			status: 401,
			want: []string{
				"planfix API error (HTTP 401) — " + loginHint + "; if it keeps failing, the account may no longer exist",
			},
		},
		{
			name:    "401 with an HTML body",
			status:  401,
			body:    "<!DOCTYPE html><html><body>Account not found</body></html>",
			want:    []string{"planfix API error (HTTP 401) — " + loginHint, "the account may no longer exist"},
			notWant: []string{"<html"},
		},
		{
			name:    "401 with code 1",
			status:  401,
			body:    `{"result":"fail","code":1,"error":"Access denied, unknown token"}`,
			want:    []string{"planfix API error (HTTP 401): Access denied, unknown token — " + loginHint},
			notWant: []string{"account may no longer exist"},
		},
		{
			name:    "401 with a PFIX_TOKEN override",
			status:  401,
			body:    `{"result":"fail","code":1,"error":"Access denied, unknown token"}`,
			env:     "bogus",
			want:    []string{"PFIX_TOKEN is set and overrides the profile's token, so correct or unset it"},
			notWant: []string{"auth login"},
		},
		{
			name:    "code 5 scope denied keeps its hint",
			status:  403,
			body:    `{"result":"fail","code":5,"error":"Scope denied, method not allowed"}`,
			want:    []string{"Scope denied, method not allowed — the token is valid but lacks the scope for this action"},
			notWant: []string{"auth login"},
		},
		{
			name:    "non-auth errors are unchanged",
			status:  400,
			body:    `{"result":"fail","code":1000,"error":"Task not found by id - 9"}`,
			want:    []string{"planfix API error (HTTP 400): Task not found by id - 9"},
			notWant: []string{" — "},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("PFIX_TOKEN", tt.env)
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			root, g := newAPIProbeRoot(srv.URL)
			root.SetArgs([]string{"--profile", "test", "apiprobe"})
			err := run(context.Background(), root, g)
			if err == nil {
				t.Fatal("want error")
			}
			msg := err.Error()
			for _, w := range tt.want {
				if !strings.Contains(msg, w) {
					t.Errorf("message %q\nmissing %q", msg, w)
				}
			}
			for _, nw := range tt.notWant {
				if strings.Contains(msg, nw) {
					t.Errorf("message %q\nshould not contain %q", msg, nw)
				}
			}
			var apiErr *planfix.APIError
			if !errors.As(err, &apiErr) {
				t.Errorf("error no longer unwraps to *planfix.APIError: %v", err)
			}
		})
	}
}

// TestRootHintWithoutProfileFlag: with no --profile flag a bare `pfix auth
// login` already targets the active profile, so the hint does not add one.
func TestRootHintWithoutProfileFlag(t *testing.T) {
	t.Setenv("PFIX_TOKEN", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
	}))
	defer srv.Close()

	root, g := newAPIProbeRoot(srv.URL)
	root.SetArgs([]string{"apiprobe"})
	err := run(context.Background(), root, g)
	if err == nil || !strings.Contains(err.Error(), "run `pfix auth login` to re-authenticate") {
		t.Errorf("err = %v, want the bare `pfix auth login` hint", err)
	}
}

// TestRootDescribesPingOnce covers a real command end to end: ping returns
// the API error as is and the root adds the hint exactly once.
func TestRootDescribesPingOnce(t *testing.T) {
	t.Setenv("PFIX_TOKEN", "")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		_, _ = io.WriteString(w, `{"result":"fail","code":1,"error":"Access denied, unknown token"}`)
	}))
	defer srv.Close()

	root, g := newRootCmd()
	root.SetOut(io.Discard)
	root.SetErr(io.Discard)
	pingCmd, _, err := root.Find([]string{"ping"})
	if err != nil {
		t.Fatal(err)
	}
	pingCmd.RunE = func(cmd *cobra.Command, _ []string) error {
		return runPing(cmd.Context(), &pingOptions{client: fakePingClient(srv.URL), out: io.Discard})
	}
	root.SetArgs([]string{"--profile", "test", "ping"})
	err = run(context.Background(), root, g)
	want := "planfix API error (HTTP 401): Access denied, unknown token — the token was rejected; run `pfix auth login --profile test` to re-authenticate"
	if err == nil || err.Error() != want {
		t.Errorf("err = %v\nwant %s", err, want)
	}
}
