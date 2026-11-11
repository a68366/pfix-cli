package file

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/a68366/pfix-cli/internal/cmdutil"
)

func TestDeleteRefusesBadInput(t *testing.T) {
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

func TestDeleteOutput(t *testing.T) {
	tests := []struct {
		name  string
		quiet bool
		json  bool
		jq    string
		want  string
		exact bool
	}{
		{"human", false, false, "", "Deleted file 12\n", true},
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
			if gotMethod != http.MethodDelete || gotPath != "/file/12" {
				t.Errorf("request = %s %s, want DELETE /file/12", gotMethod, gotPath)
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

// fileWriteErrorCases covers the error answers shared by delete and rename.
var fileWriteErrorCases = []struct {
	name   string
	status int
	body   string
	want   []string
	reject []string
}{
	{
		name:   "already deleted answers 200 with result fail",
		status: http.StatusOK,
		body:   `{"result":"fail","code":7000,"error":"File by id - 12 has already been deleted"}`,
		want:   []string{"has already been deleted", "still be viewed and downloaded"},
		reject: []string{"no file with that id"},
	},
	{
		name:   "unknown id",
		status: http.StatusBadRequest,
		body:   `{"result":"fail","code":7000,"error":"File not found by id - 12"}`,
		want:   []string{"File not found by id - 12", "no file with that id"},
		reject: []string{"still be viewed"},
	},
	{
		name:   "scope denied passes through for the root to describe",
		status: http.StatusBadRequest,
		body:   `{"result":"fail","code":5,"error":"Access denied"}`,
		want:   []string{"Access denied"},
		reject: []string{"no file with that id", "still be viewed"},
	},
}

func TestDeleteAPIErrors(t *testing.T) {
	for _, tt := range fileWriteErrorCases {
		t.Run(tt.name, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
				_, _ = io.WriteString(w, tt.body)
			}))
			defer srv.Close()

			out := &strings.Builder{}
			o := &deleteOptions{force: true, client: fakeClient(srv.URL), out: out}
			checkWriteError(t, runDelete(context.Background(), o, "12"), tt.want, tt.reject)
			if out.Len() != 0 {
				t.Errorf("nothing may be printed on failure, got %q", out.String())
			}
		})
	}
}

func checkWriteError(t *testing.T, err error, want, reject []string) {
	t.Helper()
	if err == nil {
		t.Fatal("want error")
	}
	for _, w := range want {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error = %v, want it to contain %q", err, w)
		}
	}
	for _, r := range reject {
		if strings.Contains(err.Error(), r) {
			t.Errorf("error = %v, must not contain %q", err, r)
		}
	}
}

// A 5xx must not be retried: a retry after a delete the server already
// carried out would answer "already deleted" and misreport the outcome.
func TestDeleteSendsOnce(t *testing.T) {
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

func TestFileRegistersRenameAndDelete(t *testing.T) {
	cmd := NewCmd(&cmdutil.GlobalOpts{})

	del, _, err := cmd.Find([]string{"delete"})
	if err != nil || del.Name() != "delete" {
		t.Fatalf("file delete not registered: %v", err)
	}
	for _, w := range []string{"--force", "cannot be undone", "every task, contact, and project",
		"does not erase it", "viewable and downloadable", "cannot be attached again"} {
		if !strings.Contains(del.Long, w) {
			t.Errorf("delete help missing %q: %q", w, del.Long)
		}
	}
	if del.Flags().Lookup("force") == nil {
		t.Error("delete has no --force flag")
	}

	ren, _, err := cmd.Find([]string{"rename"})
	if err != nil || ren.Name() != "rename" {
		t.Fatalf("file rename not registered: %v", err)
	}
	for _, w := range []string{"path separator", "deleted file cannot be renamed"} {
		if !strings.Contains(ren.Long, w) {
			t.Errorf("rename help missing %q: %q", w, ren.Long)
		}
	}
	if ren.Flags().Lookup("force") != nil {
		t.Error("rename is not destructive and must not take --force")
	}
}
