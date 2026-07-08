package customfield

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestTypeName(t *testing.T) {
	cases := []struct {
		in   any
		want string
	}{
		{float64(0), "Short text"},
		{float64(1), "Number"},
		{float64(16), "Task"},
		{float64(29), "Totals field"},
		{float64(30), "30"}, // unknown numeric code → raw number
		{"weird", "weird"},  // non-numeric → %v defensive
	}
	for _, c := range cases {
		if got := typeName(c.in); got != c.want {
			t.Errorf("typeName(%v) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestListDecodesType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[{"id":1,"name":"Sum","type":1},{"id":2,"name":"Mystery","type":30}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "task", client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, "Number") {
		t.Errorf("TYPE 1 not decoded to Number: %q", result)
	}
	if !strings.Contains(result, "30") {
		t.Errorf("unknown TYPE 30 not shown raw: %q", result)
	}
}

func TestListFieldsOverrideKeepsTypeRaw(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","customfields":[{"id":1,"name":"Sum","type":1}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &listOptions{objectType: "task", fields: "id,type", client: fakeClient(srv.URL), out: out}
	if err := runList(context.Background(), o); err != nil {
		t.Fatalf("runList: %v", err)
	}
	result := out.String()
	if strings.Contains(result, "Number") {
		t.Errorf("--fields override should keep TYPE raw, got decoded: %q", result)
	}
	if !strings.Contains(result, "1") {
		t.Errorf("--fields override should show raw code 1: %q", result)
	}
}
