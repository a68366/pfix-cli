package task

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunFiltersDefaultTable(t *testing.T) {
	var gotMethod, gotPath string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		io.WriteString(w, `{"result":"success","filters":[{"id":"220612","name":"Team load","owner":{"id":"user:1","name":"Pyotr"}},{"id":":all","name":"All"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &filtersOptions{client: fakeClient(srv.URL), out: out}
	if err := runFilters(context.Background(), o); err != nil {
		t.Fatalf("runFilters: %v", err)
	}
	if gotMethod != "POST" {
		t.Errorf("method = %q, want POST", gotMethod)
	}
	if gotPath != "/task/filters" {
		t.Errorf("path = %q, want /task/filters", gotPath)
	}
	result := out.String()
	for _, want := range []string{"ID", "NAME", "OWNER", "220612", "Team load", "Pyotr", ":all"} {
		if !strings.Contains(result, want) {
			t.Errorf("output missing %q: %q", want, result)
		}
	}
}

func TestRunFiltersJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","filters":[{"id":":all","name":"All"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &filtersOptions{json: true, client: fakeClient(srv.URL), out: out}
	if err := runFilters(context.Background(), o); err != nil {
		t.Fatalf("runFilters: %v", err)
	}
	result := out.String()
	if !strings.Contains(result, `"result"`) || !strings.Contains(result, `"filters"`) {
		t.Errorf("json output missing fields: %q", result)
	}
}

func TestRunFiltersQuiet(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","filters":[{"id":":in","name":"Incoming"}]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &filtersOptions{quiet: true, client: fakeClient(srv.URL), out: out}
	if err := runFilters(context.Background(), o); err != nil {
		t.Fatalf("runFilters: %v", err)
	}
	result := out.String()
	if strings.Contains(result, "ID") || strings.Contains(result, "NAME") {
		t.Errorf("quiet mode should not show headers: %q", result)
	}
	if !strings.Contains(result, "Incoming") {
		t.Errorf("quiet mode should still show data: %q", result)
	}
}

func TestRunFiltersEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"result":"success","filters":[]}`)
	}))
	defer srv.Close()

	out := &strings.Builder{}
	o := &filtersOptions{client: fakeClient(srv.URL), out: out}
	if err := runFilters(context.Background(), o); err != nil {
		t.Fatalf("runFilters: %v", err)
	}
	if !strings.Contains(out.String(), "ID") {
		t.Errorf("empty result should still render header: %q", out.String())
	}
}
