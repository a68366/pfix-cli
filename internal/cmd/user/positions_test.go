package user

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunPositionsDefaultTable(t *testing.T) {
	var gotMethod, gotPath, gotQuery string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath, gotQuery = r.Method, r.URL.Path, r.URL.RawQuery
		io.WriteString(w, `{"result":"success","positions":[{"id":10,"name":"Manager"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &positionsOptions{client: fakeClient(srv.URL), out: out}
	if err := runPositions(context.Background(), o); err != nil {
		t.Fatalf("runPositions: %v", err)
	}
	if gotMethod != "GET" {
		t.Errorf("method = %q, want GET", gotMethod)
	}
	if gotPath != "/user/positions" {
		t.Errorf("path = %q, want /user/positions", gotPath)
	}
	if !strings.Contains(gotQuery, "fields=id%2Cname") && !strings.Contains(gotQuery, "fields=id,name") {
		t.Errorf("query missing default fields: %q", gotQuery)
	}
	result := out.String()
	for _, want := range []string{"ID", "NAME", "10", "Manager"} {
		if !strings.Contains(result, want) {
			t.Errorf("output missing %q: %q", want, result)
		}
	}
}

func TestRunPositionsJSONQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","positions":[{"id":10,"name":"Manager"}]}`)
	}))
	defer srv.Close()

	jout := &strings.Builder{}
	if err := runPositions(context.Background(), &positionsOptions{json: true, client: fakeClient(srv.URL), out: jout}); err != nil {
		t.Fatalf("runPositions json: %v", err)
	}
	if !strings.Contains(jout.String(), `"positions"`) {
		t.Errorf("json output missing envelope: %q", jout.String())
	}

	qout := &strings.Builder{}
	if err := runPositions(context.Background(), &positionsOptions{quiet: true, client: fakeClient(srv.URL), out: qout}); err != nil {
		t.Fatalf("runPositions quiet: %v", err)
	}
	if strings.Contains(qout.String(), "ID") || strings.Contains(qout.String(), "NAME") {
		t.Errorf("quiet mode should not show headers: %q", qout.String())
	}
}

func TestUserCmdRegistersLookups(t *testing.T) {
	cmd := NewCmd(nil)
	have := map[string]bool{}
	for _, c := range cmd.Commands() {
		have[c.Name()] = true
	}
	for _, name := range []string{"groups", "positions"} {
		if !have[name] {
			t.Errorf("user command missing subcommand %q", name)
		}
	}
}
