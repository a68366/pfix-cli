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
		{`not json`, ""},
	}
	for _, c := range cases {
		if got := planfix.ParseError(400, []byte(c.body)).Message; got != c.want {
			t.Errorf("body %q -> %q, want %q", c.body, got, c.want)
		}
	}
}

func TestParseError_NonJSONBodyYieldsCleanMessage(t *testing.T) {
	e := planfix.ParseError(404, []byte("<html><title>404</title></html>"))
	if e.Message != "" || e.Code != 0 {
		t.Fatalf("got %+v", e)
	}
	if e.Error() != "planfix API error (HTTP 404)" {
		t.Fatalf("Error() = %q", e.Error())
	}
}

func TestParseError_ExtractsCodeAndError(t *testing.T) {
	e := planfix.ParseError(400, []byte(`{"result":"fail","code":1000,"error":"nope"}`))
	if e.Code != 1000 || e.Message != "nope" {
		t.Fatalf("got %+v", e)
	}
}
