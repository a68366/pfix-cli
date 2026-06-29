package planfix_test

import (
	"testing"

	"github.com/a68366/pfix-cli/internal/planfix"
)

func TestParseErrorMessage(t *testing.T) {
	cases := []struct{ body, want string }{
		{`{"error":"nope"}`, "nope"},
		{`{"message":"bad token"}`, "bad token"},
		{`{"error":{"message":"deep"}}`, "deep"},
		{`not json`, "not json"},
	}
	for _, c := range cases {
		if got := planfix.ParseError(400, []byte(c.body)).Message; got != c.want {
			t.Errorf("body %q -> %q, want %q", c.body, got, c.want)
		}
	}
}
