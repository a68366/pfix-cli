package cmdutil

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/planfix"
)

func TestDescribeAPIError(t *testing.T) {
	tests := []struct {
		name    string
		in      error
		ctx     AuthContext
		want    []string // substrings the message must contain
		notWant []string // substrings it must not contain
		same    bool     // the error must come back unchanged
	}{
		{
			name: "code 1 unknown token adds a re-login hint",
			in:   planfix.ParseError(401, []byte(`{"result":"fail","code":1,"error":"Access denied, unknown token"}`)),
			want: []string{
				"Access denied, unknown token",
				"the token was rejected; run `pfix auth login` to re-authenticate",
			},
			notWant: []string{"account may no longer exist"},
		},
		{
			name: "code-less 401 with an empty body still points at auth login",
			in:   planfix.ParseError(401, nil),
			want: []string{
				"planfix API error (HTTP 401) — the token was rejected; run `pfix auth login` to re-authenticate",
				"if it keeps failing, the account may no longer exist",
			},
		},
		{
			name: "code-less 401 with an HTML body still points at auth login",
			in:   planfix.ParseError(401, []byte("<!DOCTYPE html><html><body>No account</body></html>")),
			want: []string{
				"planfix API error (HTTP 401) — the token was rejected",
				"the account may no longer exist",
			},
			notWant: []string{"<html"},
		},
		{
			name: "the --profile flag is named in the login command",
			in:   planfix.ParseError(401, nil),
			ctx:  AuthContext{Profile: "test"},
			want: []string{"run `pfix auth login --profile test` to re-authenticate"},
		},
		{
			name: "a PFIX_TOKEN override is named instead of auth login",
			in:   planfix.ParseError(401, []byte(`{"result":"fail","code":1,"error":"Access denied, unknown token"}`)),
			ctx:  AuthContext{Profile: "test", TokenFromEnv: true},
			want: []string{
				"the token was rejected; PFIX_TOKEN is set and overrides the profile's token, so correct or unset it",
			},
			notWant: []string{"auth login"},
		},
		{
			name:    "code 5 scope denied explains the token is valid",
			in:      planfix.ParseError(405, []byte(`{"result":"fail","code":5,"error":"Scope denied, method not allowed"}`)),
			ctx:     AuthContext{Profile: "test"},
			want:    []string{"Scope denied", "— the token is valid but lacks the scope for this action"},
			notWant: []string{"auth login"},
		},
		{
			name: "other app codes pass through unchanged",
			in:   planfix.ParseError(400, []byte(`{"result":"fail","code":1000,"error":"Task not found by id - 9"}`)),
			same: true,
		},
		{
			name: "a code-less non-401 error passes through unchanged",
			in:   planfix.ParseError(403, nil),
			same: true,
		},
		{
			name: "a code-less 500 passes through unchanged",
			in:   planfix.ParseError(500, []byte("<html>oops</html>")),
			same: true,
		},
		{
			name: "non-API errors pass through unchanged",
			in:   errors.New("connection refused"),
			same: true,
		},
		{
			name: "nil passes through",
			in:   nil,
			same: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DescribeAPIError(tt.in, tt.ctx)
			if tt.same {
				if got != tt.in {
					t.Fatalf("error was modified: %v", got)
				}
				return
			}
			msg := got.Error()
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
			if !errors.As(got, &apiErr) {
				t.Errorf("hinted error no longer unwraps to *planfix.APIError")
			}
		})
	}
}

// TestDescribeAPIErrorIsIdempotent: describing an already-described error,
// even one wrapped since, must not add the hint a second time.
func TestDescribeAPIErrorIsIdempotent(t *testing.T) {
	for _, body := range []string{"", `{"result":"fail","code":1,"error":"Access denied, unknown token"}`} {
		ctx := AuthContext{Profile: "test"}
		wrapped := fmt.Errorf("resolve comment: %w", DescribeAPIError(planfix.ParseError(401, []byte(body)), ctx))
		got := DescribeAPIError(wrapped, ctx)
		if got != wrapped {
			t.Errorf("body %q: described error was modified: %v", body, got)
		}
		if n := strings.Count(got.Error(), "the token was rejected"); n != 1 {
			t.Errorf("body %q: hint appears %d times: %q", body, n, got)
		}
	}
}

// TestDescribeAPIErrorKeepsCommandHint: a command-specific hint wrapped around
// an auth-free API error is left alone by the root pass.
func TestDescribeAPIErrorKeepsCommandHint(t *testing.T) {
	in := fmt.Errorf("%w — no task with that id", planfix.ParseError(400, []byte(`{"result":"fail","code":1000,"error":"x"}`)))
	if got := DescribeAPIError(in, AuthContext{Profile: "test"}); got != in {
		t.Errorf("command hint was modified: %v", got)
	}
}

// TestDescribeAPIErrorSkipsMarkedErrors: an error marked WithoutAuthHint is
// returned unchanged, even when it wraps an auth failure.
func TestDescribeAPIErrorSkipsMarkedErrors(t *testing.T) {
	if WithoutAuthHint(nil) != nil {
		t.Error("WithoutAuthHint(nil) should be nil")
	}
	for _, body := range []string{"", `{"result":"fail","code":1,"error":"Access denied, unknown token"}`} {
		in := WithoutAuthHint(fmt.Errorf("credential check failed: %w", planfix.ParseError(401, []byte(body))))
		got := DescribeAPIError(in, AuthContext{Profile: "test", TokenFromEnv: true})
		if got != in {
			t.Errorf("body %q: marked error was modified: %v", body, got)
		}
		var apiErr *planfix.APIError
		if !errors.As(got, &apiErr) {
			t.Errorf("body %q: marked error no longer unwraps to *planfix.APIError", body)
		}
	}
}
