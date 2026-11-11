package task

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func TestRunDeleteRefusesBadInput(t *testing.T) {
	tests := []struct {
		name    string
		force   bool
		id      string
		wantErr string
	}{
		{"no force", false, "12", "--force"},
		{"bad id", true, "abc", "abc"},
		{"zero id", true, "0", "0"},
		{"negative id", true, "-3", "-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hit := false
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				hit = true
			}))
			defer srv.Close()

			o := &deleteOptions{force: tt.force, client: fakeClient(srv.URL), out: &strings.Builder{}}
			err := runDelete(context.Background(), o, tt.id)
			if err == nil {
				t.Fatal("want error")
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("error = %v, want it to mention %q", err, tt.wantErr)
			}
			if hit {
				t.Error("no request may be sent")
			}
		})
	}
}

func TestRunDeleteOutput(t *testing.T) {
	tests := []struct {
		name  string
		quiet bool
		json  bool
		jq    string
		want  string
		exact bool
	}{
		{"human", false, false, "", "Deleted task 12\n", true},
		{"quiet", true, false, "", "12\n", true},
		{"json", false, true, "", `"result": "success"`, false},
		{"jq", false, true, ".result", "success\n", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotMethod, gotPath string
			var gotBody []byte
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotMethod, gotPath = r.Method, r.URL.Path
				gotBody, _ = io.ReadAll(r.Body)
				_, _ = io.WriteString(w, `{"result":"success"}`)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &deleteOptions{
				force: true, quiet: tt.quiet, json: tt.json, jq: tt.jq,
				client: fakeClient(srv.URL), out: out,
			}
			if err := runDelete(context.Background(), o, "12"); err != nil {
				t.Fatalf("runDelete: %v", err)
			}
			if gotMethod != http.MethodDelete || gotPath != "/task/12" {
				t.Errorf("request = %s %s, want DELETE /task/12", gotMethod, gotPath)
			}
			if len(gotBody) != 0 {
				t.Errorf("body = %q, want none", gotBody)
			}
			if tt.exact && out.String() != tt.want {
				t.Errorf("output = %q, want %q", out.String(), tt.want)
			}
			if !tt.exact && !strings.Contains(out.String(), tt.want) {
				t.Errorf("output = %q, want it to contain %q", out.String(), tt.want)
			}
		})
	}
}

func TestRunDeleteAPIErrors(t *testing.T) {
	tests := []struct {
		name   string
		status int
		body   string
		want   []string
		reject []string
	}{
		{
			name:   "unknown or already deleted task",
			status: http.StatusBadRequest,
			body:   `{"result":"fail","code":1000,"error":"Task not found by id - 12"}`,
			want:   []string{"Task not found by id - 12", "already deleted"},
		},
		{
			name:   "200 with result fail is an error",
			status: http.StatusOK,
			body:   `{"result":"fail","code":1000,"error":"Task not found by id - 12"}`,
			want:   []string{"Task not found by id - 12", "already deleted"},
		},
		{
			name:   "scope denied passes through for the root to describe",
			status: http.StatusBadRequest,
			body:   `{"result":"fail","code":5,"error":"Access denied"}`,
			want:   []string{"Access denied"},
			reject: []string{"already deleted"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &deleteOptions{force: true, client: fakeClient(srv.URL), out: out}
			err := runDelete(context.Background(), o, "12")
			if err == nil {
				t.Fatal("want error")
			}
			for _, w := range tt.want {
				if !strings.Contains(err.Error(), w) {
					t.Errorf("error = %v, want it to contain %q", err, w)
				}
			}
			for _, r := range tt.reject {
				if strings.Contains(err.Error(), r) {
					t.Errorf("error = %v, must not contain %q", err, r)
				}
			}
			if out.Len() != 0 {
				t.Errorf("nothing may be printed on failure, got %q", out.String())
			}
		})
	}
}

// A 5xx must not be retried: a retry after a delete the server already
// carried out would answer "not found" and misreport the outcome.
func TestRunDeleteSendsOnce(t *testing.T) {
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusBadGateway)
	}))
	defer srv.Close()

	o := &deleteOptions{force: true, client: fakeClient(srv.URL), out: &strings.Builder{}}
	if err := runDelete(context.Background(), o, "12"); err == nil {
		t.Fatal("want error")
	}
	if hits != 1 {
		t.Errorf("requests = %d, want 1", hits)
	}
}

func TestTaskRegistersDelete(t *testing.T) {
	cmd := NewCmd(&cmdutil.GlobalOpts{})
	sub, _, err := cmd.Find([]string{"delete"})
	if err != nil || sub.Name() != "delete" {
		t.Fatalf("task delete not registered: %v", err)
	}
	for _, w := range []string{"--force", "cannot be undone", "checklist", "subtasks are NOT deleted", "top-level", "comments and attached files"} {
		if !strings.Contains(sub.Long, w) {
			t.Errorf("delete help missing %q: %q", w, sub.Long)
		}
	}
	if sub.Flags().Lookup("force") == nil {
		t.Error("delete has no --force flag")
	}
}
