package output

import (
	"bytes"
	"strings"
	"testing"
)

func TestCompileJQValid(t *testing.T) {
	q, err := CompileJQ(".tasks[].name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q == nil {
		t.Fatal("expected a non-nil query")
	}
}

func TestCompileJQInvalid(t *testing.T) {
	_, err := CompileJQ(".[")
	if err == nil {
		t.Fatal("expected an error for a malformed expression")
	}
	if !strings.Contains(err.Error(), "invalid --jq expression") {
		t.Fatalf("error %q does not mention the expected context", err)
	}
}

func TestEmitJSONEmptyExprDelegatesToJSON(t *testing.T) {
	cases := []string{`{"a":1}`, "plain text"}
	for _, raw := range cases {
		var got, want bytes.Buffer
		if err := EmitJSON(&got, []byte(raw), ""); err != nil {
			t.Fatalf("EmitJSON(%q) unexpected error: %v", raw, err)
		}
		if err := JSON(&want, []byte(raw)); err != nil {
			t.Fatalf("JSON(%q) unexpected error: %v", raw, err)
		}
		if got.String() != want.String() {
			t.Fatalf("EmitJSON(%q) = %q, want %q", raw, got.String(), want.String())
		}
	}
}

func TestEmitJSONFiltered(t *testing.T) {
	cases := []struct {
		name string
		raw  string
		expr string
		want string
	}{
		{"identity object", `{"a":1}`, ".", "{\"a\":1}\n"},
		{"string array projection", `{"tasks":[{"name":"a"},{"name":"b"}]}`, ".tasks[].name", "a\nb\n"},
		{"string result unquoted", `{"name":"hi"}`, ".name", "hi\n"},
		{"numeric result", `{"n":42}`, ".n", "42\n"},
		{"object result compact", `{"tasks":[{"name":"a"}]}`, ".tasks[0]", "{\"name\":\"a\"}\n"},
		{"missing key yields null", `{}`, ".missing", "null\n"},
		{"empty result set", `{"tasks":[]}`, ".tasks[]", ""},
		// Large integer ids (beyond 2^53) must survive exactly, not round to a
		// float. 123456789012345678 would become 123456789012345680 under a
		// float64 decode; json.Number keeps it intact.
		{"large integer id preserved", `{"task":{"id":123456789012345678}}`, ".task.id", "123456789012345678\n"},
		{"large integer arithmetic exact", `{"task":{"id":123456789012345678}}`, ".task.id + 1", "123456789012345679\n"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			var b bytes.Buffer
			if err := EmitJSON(&b, []byte(c.raw), c.expr); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if b.String() != c.want {
				t.Fatalf("EmitJSON(%q, %q) = %q, want %q", c.raw, c.expr, b.String(), c.want)
			}
		})
	}
}

func TestEmitJSONRuntimeError(t *testing.T) {
	var b bytes.Buffer
	err := EmitJSON(&b, []byte("[]"), ".foo")
	if err == nil {
		t.Fatal("expected a runtime error indexing an array with a key")
	}
	if !strings.Contains(err.Error(), "--jq error") {
		t.Fatalf("error %q does not mention the expected context", err)
	}
}

func TestEmitJSONInvalidJQExpr(t *testing.T) {
	var b bytes.Buffer
	err := EmitJSON(&b, []byte(`{"a":1}`), ".[")
	if err == nil {
		t.Fatal("expected a compile error")
	}
	if !strings.Contains(err.Error(), "invalid --jq expression") {
		t.Fatalf("error %q does not mention the expected context", err)
	}
}

func TestEmitJSONNonJSONInputWithExpr(t *testing.T) {
	var b bytes.Buffer
	err := EmitJSON(&b, []byte("not json"), ".")
	if err == nil {
		t.Fatal("expected an error for non-JSON input")
	}
	if !strings.Contains(err.Error(), "response is not valid JSON") {
		t.Fatalf("error %q does not mention the expected context", err)
	}
}
